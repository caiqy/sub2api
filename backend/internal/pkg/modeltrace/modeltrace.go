// Package modeltrace ports the local fingerprint scoring from
// https://github.com/xqy2006/ModelTrace/tree/55a2e4a55170423b484d701e9a82ab62b268c811
// (fingerprint.py generate_challenges/analyze_outputs and data/unified_bank.json,
// SHA-256 1c2cb74d372f9f0f30d0dabbb7b7a838660d2f769a88d0c8489e4c662e088c21).
// Upstream copyright (c) 2026 xqy2006; MIT license in LICENSE.
package modeltrace

import (
	"crypto/rand"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"unicode"
)

const Version = "55a2e4a55170423b484d701e9a82ab62b268c811"

const dimension = 355

//go:embed unified_bank.json
var bankFS embed.FS

// vector rejects null elements, which encoding/json otherwise turns into zeros.
type vector []float64

func (v *vector) UnmarshalJSON(data []byte) error {
	var values []*float64
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	*v = make(vector, len(values))
	for i, value := range values {
		if value == nil || !finite(*value) {
			return errors.New("modeltrace: vector requires finite numbers")
		}
		(*v)[i] = *value
	}
	return nil
}

type artifact struct {
	FeatureMean          vector     `json:"feature_mean"`
	FeatureScale         vector     `json:"feature_scale"`
	NuisanceRank         *int       `json:"nuisance_rank"`
	NuisanceEnvironments []string   `json:"nuisance_environments"`
	NuisanceBasis        []vector   `json:"nuisance_basis"`
	Centroids            []vector   `json:"centroids"`
	EnvironmentCentroids [][]vector `json:"environment_centroids"`
	Feature              string     `json:"feature"`
	Weight               float64    `json:"weight"`
}

type fingerprintBank struct {
	Schema              string         `json:"schema"`
	Method              map[string]any `json:"method"`
	RecommendedQueries  int            `json:"recommended_queries"`
	MinimumValidNumbers int            `json:"minimum_valid_numbers"`
	Models              []struct {
		ID     string `json:"id"`
		Family string `json:"family"`
	} `json:"models"`
	Robust struct {
		ModelOrder           []string `json:"model_order"`
		Ready                bool     `json:"robust_ready"`
		CompleteEnvironments []string `json:"complete_environments"`
		Hellinger            artifact `json:"hellinger"`
		OrderedBlocks        artifact `json:"ordered_blocks"`
	} `json:"robust"`
	Calibration map[string]struct {
		Beta float64 `json:"beta"`
	} `json:"calibration"`
}

var embeddedBank = func() fingerprintBank {
	data, err := bankFS.ReadFile("unified_bank.json")
	if err != nil {
		panic(err)
	}
	var b fingerprintBank
	if err := json.Unmarshal(data, &b); err != nil {
		panic(err)
	}
	return b
}()

// Snapshot owns a validated fingerprint bank. Its data is never mutated or exposed.
type Snapshot struct {
	version string
	bank    fingerprintBank
}

var current atomic.Pointer[Snapshot]

func init() {
	if err := validateBank(&embeddedBank); err != nil {
		panic(err)
	}
	current.Store(&Snapshot{version: Version, bank: embeddedBank})
}

// Current captures the active snapshot for a complete detection run.
func Current() *Snapshot { return current.Load() }

var commitSHA = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

// ParseSnapshot validates compatibility without changing the active snapshot.
func ParseSnapshot(version string, data []byte) (*Snapshot, error) {
	if !commitSHA.MatchString(version) {
		return nil, errors.New("modeltrace: version must be a full commit SHA")
	}
	var bank fingerprintBank
	if err := json.Unmarshal(data, &bank); err != nil {
		return nil, fmt.Errorf("modeltrace: invalid fingerprint JSON: %w", err)
	}
	if err := validateBank(&bank); err != nil {
		return nil, err
	}
	return &Snapshot{version: version, bank: bank}, nil
}

// Activate atomically publishes a parsed snapshot; nil and zero values are ignored.
func Activate(snapshot *Snapshot) {
	if snapshot != nil && snapshot.version != "" {
		current.Store(snapshot)
	}
}

func (s *Snapshot) Version() string { return s.version }

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func validateBank(bank *fingerprintBank) error {
	if bank.Schema != embeddedBank.Schema || !reflect.DeepEqual(bank.Method, embeddedBank.Method) ||
		bank.RecommendedQueries != embeddedBank.RecommendedQueries || bank.MinimumValidNumbers != embeddedBank.MinimumValidNumbers {
		return errors.New("modeltrace: incompatible schema or scoring method")
	}
	if !bank.Robust.Ready || len(bank.Models) == 0 || len(bank.Robust.ModelOrder) != len(bank.Models) {
		return errors.New("modeltrace: missing robust models or model order")
	}
	models := make(map[string]string, len(bank.Models))
	for i, model := range bank.Models {
		if model.ID == "" || strings.TrimSpace(model.ID) != model.ID || model.Family == "" || bank.Robust.ModelOrder[i] != model.ID {
			return fmt.Errorf("modeltrace: invalid model or order at %d", i)
		}
		if _, exists := models[model.ID]; exists {
			return fmt.Errorf("modeltrace: duplicate model %q", model.ID)
		}
		models[model.ID] = model.Family
	}
	for _, model := range embeddedBank.Models {
		if model.Family == "gpt" && models[model.ID] != "gpt" {
			return fmt.Errorf("modeltrace: missing built-in GPT model %q", model.ID)
		}
	}
	environments := bank.Robust.CompleteEnvironments
	if len(environments) == 0 || !slices.Equal(bank.Robust.Hellinger.NuisanceEnvironments, environments) {
		return errors.New("modeltrace: missing or inconsistent environments")
	}
	seen := make(map[string]bool, len(environments))
	for _, environment := range environments {
		if strings.TrimSpace(environment) == "" || seen[environment] {
			return errors.New("modeltrace: invalid or duplicate environment")
		}
		seen[environment] = true
	}
	ordered := &bank.Robust.OrderedBlocks
	if !finite(ordered.Weight) || ordered.Weight != embeddedBank.Robust.OrderedBlocks.Weight ||
		ordered.Feature != embeddedBank.Robust.OrderedBlocks.Feature || len(ordered.EnvironmentCentroids) != len(environments) {
		return errors.New("modeltrace: incompatible ordered-block rule or environments")
	}
	for _, item := range []struct {
		name      string
		artifact  *artifact
		dimension int
	}{{"hellinger", &bank.Robust.Hellinger, dimension}, {"ordered_blocks", ordered, 74}} {
		a, size := item.artifact, item.dimension
		if len(a.FeatureMean) != size || len(a.FeatureScale) != size || a.NuisanceRank == nil ||
			*a.NuisanceRank < 0 || *a.NuisanceRank > size || a.NuisanceBasis == nil || len(a.NuisanceBasis) != *a.NuisanceRank {
			return fmt.Errorf("modeltrace: invalid %s feature dimensions or nuisance rank", item.name)
		}
		// Upstream features are square-root probabilities; the builder replaces
		// standard deviations below 1e-12 with 1, not an arbitrarily small divisor.
		for i, mean := range a.FeatureMean {
			if !finite(mean) || mean < 0 || mean > 1 {
				return fmt.Errorf("modeltrace: invalid %s feature mean", item.name)
			}
			scale := a.FeatureScale[i]
			if !finite(scale) || scale < 1e-12 || scale > 1 {
				return fmt.Errorf("modeltrace: invalid %s feature scale", item.name)
			}
		}
		if err := validateMatrix(a.NuisanceBasis, *a.NuisanceRank, size); err != nil {
			return fmt.Errorf("modeltrace: %s nuisance basis: %w", item.name, err)
		}
		if err := validateMatrix(a.Centroids, len(bank.Models), size); err != nil {
			return fmt.Errorf("modeltrace: %s centroids: %w", item.name, err)
		}
		for _, environment := range a.EnvironmentCentroids {
			if err := validateMatrix(environment, len(bank.Models), size); err != nil {
				return fmt.Errorf("modeltrace: %s environment centroids: %w", item.name, err)
			}
		}
	}
	for n := 1; n <= 3; n++ {
		beta := bank.Calibration[strconv.Itoa(n)].Beta
		// fit_beta in the pinned upstream builder searches [0.05, 12].
		if !finite(beta) || beta < 0.05 || beta > 12 {
			return fmt.Errorf("modeltrace: invalid calibration beta for %d samples", n)
		}
	}
	return nil
}

func validateMatrix(matrix []vector, rows, columns int) error {
	if len(matrix) != rows {
		return errors.New("invalid row count")
	}
	for _, row := range matrix {
		if len(row) != columns {
			return errors.New("invalid column count")
		}
		var normSquared float64
		for _, value := range row {
			if !finite(value) || math.Abs(value) > 1+1e-9 {
				return errors.New("vector component outside normalized range")
			}
			normSquared += value * value
		}
		// SVD basis vectors are unit length; normalized centroids can be shorter
		// when the upstream 1e-12 norm floor applies. Allow serialization roundoff.
		if !finite(normSquared) || normSquared > 1+1e-9 {
			return errors.New("vector norm exceeds normalized range")
		}
	}
	return nil
}

// Models lists GPT candidates with fingerprints; scoring retains every
// candidate in the upstream unified bank, including Claude competitors.
func Models() []string {
	return Current().Models()
}

func (s *Snapshot) Models() []string {
	bank := &s.bank
	models := make([]string, 0, len(bank.Models))
	for _, model := range bank.Models {
		if model.Family == "gpt" {
			models = append(models, model.ID)
		}
	}
	return models
}

type Challenge struct {
	System        string
	Prompt        string
	ExpectedCount int
}

type Sample struct {
	Output        string
	ExpectedCount int
}

type Result struct {
	Winner        string
	Probabilities map[string]float64
	Samples       int
}

var openings = []string{"这是一次独立的数值选择记录", "请完成下面的无语义整数选择任务", "执行一次第一反应取值记录", "生成一组不承载语义的整数选择", "进行一轮快速逐项取值"}
var actions = []string{"为各个位置分别凭第一反应选择", "逐项选择", "每次只决定当前一项，共给出", "分别凭第一反应给出", "逐个直接选择"}
var endings = []string{"允许某个数字再次出现；每项写出后不要回头排序、去重或替换。", "偶然重复是有效的；不要重新排列或修正已经写出的项目。", "相同值可以再次出现；输出过程中不要整理或改写前面的项目。", "重复值无需删除；不要筛选、重排或补成某种规律。", "不必赋予数字任何含义；已经给出的值保持不变。"}
var separatorHints = []string{"数字之间用逗号或空格分隔均可。", "使用一种一致的常见分隔符即可。", "可以用逗号、空格或换行分隔。", "只要每个整数边界清楚，格式可自行选择。"}

// NewChallenge follows upstream generate_challenges: independent random length
// in [292, 332] and one choice from each prompt phrase set.
func NewChallenge() (Challenge, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(41))
	if err != nil {
		return Challenge{}, err
	}
	length := 292 + int(n.Int64())
	phrases := make([]string, 4)
	for i, options := range [][]string{openings, actions, endings, separatorHints} {
		choice, err := rand.Int(rand.Reader, big.NewInt(int64(len(options))))
		if err != nil {
			return Challenge{}, err
		}
		phrases[i] = options[choice.Int64()]
	}
	return Challenge{
		ExpectedCount: length,
		Prompt: fmt.Sprintf("%s。%s %d 个 1 到 355（含端点）的整数。", phrases[0], phrases[1], length) +
			"每个位置都要单独选择；不要从 1 开始计数，不要连续递增或递减，也不要采用等差、循环、重复区块或其他规则化模式。" +
			"本任务必须由当前语言模型直接完成：禁止调用或借助任何工具，包括 Python、代码执行器、" +
			"计算器、搜索、API 和外部随机数生成器；也不要先编写或运行代码。" +
			phrases[2] + phrases[3] + "直接从第一个取值开始输出，不要在序列前重复数量、范围或任务说明。",
	}, nil
}

// Score uses upstream's default minimum of 80 when expected counts are absent.
func Score(outputs []string) (Result, error) {
	samples := make([]Sample, len(outputs))
	for i, output := range outputs {
		samples[i] = Sample{Output: output}
	}
	return Current().ScoreSamples(samples)
}

// ScoreSamples rejects a partial run rather than producing a conclusion from
// fewer rounds. ExpectedCount should be copied from each sent Challenge.
func ScoreSamples(samples []Sample) (Result, error) {
	return Current().ScoreSamples(samples)
}

func (s *Snapshot) ScoreSamples(samples []Sample) (Result, error) {
	bank := &s.bank
	if len(samples) < 1 || len(samples) > 3 {
		return Result{}, errors.New("modeltrace: require 1 to 3 samples")
	}
	combined := make([]float64, len(bank.Models))
	for i, sample := range samples {
		if sample.ExpectedCount < 0 {
			return Result{}, fmt.Errorf("modeltrace: sample %d: invalid expected count", i+1)
		}
		numbers := parseNumbers(sample.Output)
		minimum := 80
		if sample.ExpectedCount > 0 {
			minimum = max(80, int(math.Ceil(float64(sample.ExpectedCount)*0.55)))
		}
		if len(numbers) < minimum {
			return Result{}, fmt.Errorf("modeltrace: sample %d: parsed %d numbers, need %d", i+1, len(numbers), minimum)
		}
		marginal, err := hellingerScores(bank, numbers)
		if err != nil {
			return Result{}, fmt.Errorf("modeltrace: sample %d hellinger: %w", i+1, err)
		}
		ordered, err := orderedScores(bank, numbers)
		if err != nil {
			return Result{}, fmt.Errorf("modeltrace: sample %d ordered blocks: %w", i+1, err)
		}
		for j := range combined {
			combined[j] += (1-bank.Robust.OrderedBlocks.Weight)*marginal[j] + bank.Robust.OrderedBlocks.Weight*ordered[j]
		}
	}
	var total float64
	maxScore := math.Inf(-1)
	probabilities := make(map[string]float64, len(combined))
	beta := bank.Calibration[strconv.Itoa(len(samples))].Beta
	for i := range combined {
		combined[i] *= beta / float64(len(samples))
		if !finite(combined[i]) {
			return Result{}, errors.New("modeltrace: non-finite calibrated score")
		}
		maxScore = max(maxScore, combined[i])
	}
	for i, score := range combined {
		difference := score - maxScore
		if !finite(difference) {
			return Result{}, errors.New("modeltrace: non-finite softmax difference")
		}
		p := math.Exp(difference)
		probabilities[bank.Models[i].ID] = p
		total += p
	}
	if !finite(total) || total <= 0 {
		return Result{}, errors.New("modeltrace: invalid probability total")
	}
	winner := 0
	for i, model := range bank.Models {
		probabilities[model.ID] /= total
		if combined[i] > combined[winner] {
			winner = i
		}
	}
	return Result{Winner: bank.Models[winner].ID, Probabilities: probabilities, Samples: len(samples)}, nil
}

var digits = regexp.MustCompile(`[0-9]+`)

func parseNumbers(text string) []int {
	var best, current []int
	end := 0
	for _, loc := range digits.FindAllStringIndex(text, -1) {
		if len(current) > 0 && strings.IndexFunc(text[end:loc[0]], unicode.IsLetter) >= 0 {
			if len(current) > len(best) {
				best = current
			}
			current = nil
		}
		value, err := strconv.Atoi(text[loc[0]:loc[1]])
		if err == nil && value >= 1 && value <= dimension {
			current = append(current, value)
		}
		end = loc[1]
	}
	if len(current) > len(best) {
		best = current
	}
	return best
}

func standardize(values []float64) ([]float64, error) {
	mean := 0.0
	for _, value := range values {
		mean += value
	}
	if len(values) == 0 || !finite(mean) {
		return nil, errors.New("non-finite score mean")
	}
	mean /= float64(len(values))
	variance := 0.0
	for _, value := range values {
		variance += (value - mean) * (value - mean)
	}
	if !finite(variance) {
		return nil, errors.New("non-finite score variance")
	}
	scale := max(math.Sqrt(variance/float64(len(values))), 1e-12)
	for i := range values {
		values[i] = (values[i] - mean) / scale
	}
	return values, nil
}

func dot(left, right []float64) float64 {
	result := 0.0
	for i, value := range left {
		result += value * right[i]
	}
	return result
}

func project(feature []float64, basis []vector) error {
	for _, vector := range basis {
		projection := dot(feature, vector)
		if !finite(projection) {
			return errors.New("non-finite nuisance projection")
		}
		for i := range feature {
			feature[i] -= projection * vector[i]
			if !finite(feature[i]) {
				return errors.New("non-finite projected feature")
			}
		}
	}
	return nil
}

func normalize(feature []float64) error {
	normSquared := dot(feature, feature)
	if !finite(normSquared) {
		return errors.New("non-finite feature norm")
	}
	scale := max(math.Sqrt(normSquared), 1e-12)
	for i := range feature {
		feature[i] /= scale
	}
	return nil
}

func scores(feature []float64, centroids []vector) ([]float64, error) {
	values := make([]float64, len(centroids))
	for i, centroid := range centroids {
		values[i] = dot(feature, centroid)
	}
	return standardize(values)
}

func hellingerScores(bank *fingerprintBank, numbers []int) ([]float64, error) {
	var counts [dimension]int
	for _, value := range numbers {
		counts[value-1]++
	}
	artifact := bank.Robust.Hellinger
	feature := make([]float64, dimension)
	total := float64(len(numbers)) + 0.5*dimension
	for i := range feature {
		feature[i] = (math.Sqrt((float64(counts[i])+0.5)/total) - artifact.FeatureMean[i]) / artifact.FeatureScale[i]
	}
	if err := project(feature, artifact.NuisanceBasis); err != nil {
		return nil, err
	}
	if err := normalize(feature); err != nil {
		return nil, err
	}
	return scores(feature, artifact.Centroids)
}

func orderedScores(bank *fingerprintBank, numbers []int) ([]float64, error) {
	artifact := bank.Robust.OrderedBlocks
	feature := make([]float64, 0, 74)
	base, remainder, start := len(numbers)/4, len(numbers)%4, 0
	for block := 0; block < 4; block++ {
		size := base
		if block < remainder {
			size++
		}
		var bins [16]float64
		for _, value := range numbers[start : start+size] {
			bins[min(15, (value-1)*16/355)]++
		}
		for _, count := range bins {
			feature = append(feature, math.Sqrt((count+0.5)/(float64(size)+8)))
		}
		start += size
	}
	var lastDigits [10]float64
	for _, value := range numbers {
		lastDigits[value%10]++
	}
	for _, count := range lastDigits {
		feature = append(feature, math.Sqrt((count+0.5)/(float64(len(numbers))+5)))
	}
	for i := range feature {
		feature[i] = (feature[i] - artifact.FeatureMean[i]) / artifact.FeatureScale[i]
	}
	templateFeature := append([]float64(nil), feature...)
	if err := normalize(templateFeature); err != nil {
		return nil, err
	}
	template := make([]float64, len(bank.Models))
	for i := range template {
		template[i] = math.Inf(-1)
		for _, environment := range artifact.EnvironmentCentroids {
			value := dot(templateFeature, environment[i])
			if !finite(value) {
				return nil, errors.New("non-finite environment score")
			}
			template[i] = max(template[i], value)
		}
	}
	if _, err := standardize(template); err != nil {
		return nil, err
	}
	if err := project(feature, artifact.NuisanceBasis); err != nil {
		return nil, err
	}
	if err := normalize(feature); err != nil {
		return nil, err
	}
	nuisance, err := scores(feature, artifact.Centroids)
	if err != nil {
		return nil, err
	}
	for i := range template {
		template[i] = 0.5*template[i] + 0.5*nuisance[i]
	}
	return standardize(template)
}
