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
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

const Version = "55a2e4a55170423b484d701e9a82ab62b268c811"

const dimension = 355

//go:embed unified_bank.json
var bankFS embed.FS

type artifact struct {
	FeatureMean          []float64     `json:"feature_mean"`
	FeatureScale         []float64     `json:"feature_scale"`
	NuisanceBasis        [][]float64   `json:"nuisance_basis"`
	Centroids            [][]float64   `json:"centroids"`
	EnvironmentCentroids [][][]float64 `json:"environment_centroids"`
	Weight               float64       `json:"weight"`
}

type fingerprintBank struct {
	Models []struct {
		ID     string `json:"id"`
		Family string `json:"family"`
	} `json:"models"`
	Robust struct {
		Hellinger     artifact `json:"hellinger"`
		OrderedBlocks artifact `json:"ordered_blocks"`
	} `json:"robust"`
	Calibration map[string]struct {
		Beta float64 `json:"beta"`
	} `json:"calibration"`
}

var bank = func() fingerprintBank {
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

// Models lists GPT candidates with fingerprints; scoring retains every
// candidate in the upstream unified bank, including Claude competitors.
func Models() []string {
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
	return ScoreSamples(samples)
}

// ScoreSamples rejects a partial run rather than producing a conclusion from
// fewer rounds. ExpectedCount should be copied from each sent Challenge.
func ScoreSamples(samples []Sample) (Result, error) {
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
		marginal := hellingerScores(numbers)
		ordered := orderedScores(numbers)
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
		maxScore = max(maxScore, combined[i])
	}
	for i, score := range combined {
		p := math.Exp(score - maxScore)
		probabilities[bank.Models[i].ID] = p
		total += p
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

func standardize(values []float64) []float64 {
	mean := 0.0
	for _, value := range values {
		mean += value
	}
	mean /= float64(len(values))
	variance := 0.0
	for _, value := range values {
		variance += (value - mean) * (value - mean)
	}
	scale := max(math.Sqrt(variance/float64(len(values))), 1e-12)
	for i := range values {
		values[i] = (values[i] - mean) / scale
	}
	return values
}

func dot(left, right []float64) float64 {
	result := 0.0
	for i, value := range left {
		result += value * right[i]
	}
	return result
}

func project(feature []float64, basis [][]float64) {
	for _, vector := range basis {
		projection := dot(feature, vector)
		for i := range feature {
			feature[i] -= projection * vector[i]
		}
	}
}

func normalize(feature []float64) {
	scale := max(math.Sqrt(dot(feature, feature)), 1e-12)
	for i := range feature {
		feature[i] /= scale
	}
}

func scores(feature []float64, centroids [][]float64) []float64 {
	values := make([]float64, len(centroids))
	for i, centroid := range centroids {
		values[i] = dot(feature, centroid)
	}
	return standardize(values)
}

func hellingerScores(numbers []int) []float64 {
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
	project(feature, artifact.NuisanceBasis)
	normalize(feature)
	return scores(feature, artifact.Centroids)
}

func orderedScores(numbers []int) []float64 {
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
	normalize(templateFeature)
	template := make([]float64, len(bank.Models))
	for i := range template {
		template[i] = math.Inf(-1)
		for _, environment := range artifact.EnvironmentCentroids {
			template[i] = max(template[i], dot(templateFeature, environment[i]))
		}
	}
	standardize(template)
	project(feature, artifact.NuisanceBasis)
	normalize(feature)
	nuisance := scores(feature, artifact.Centroids)
	for i := range template {
		template[i] = 0.5*template[i] + 0.5*nuisance[i]
	}
	return standardize(template)
}
