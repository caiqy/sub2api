package modeltrace

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// gpt_reference.jsonl, gpt-6-astra__query-01/02/03 at Version.
// Expected probabilities are from upstream static/fingerprint-core.js
// analyzeGlobalOutputs using those rows and data/unified_bank.json.
var astraReference = []Sample{
	{ExpectedCount: 218, Output: `[247,83,319,146,28,204,337,112,59,271,164,352,91,223,17,308,135,76,289,192,43,258,121,334,68,215,301,154,9,237,178,325,54,109,283,196,31,342,87,261,143,22,310,185,73,229,348,116,294,48,203,157,331,95,266,12,181,316,64,240,127,354,39,218,172,298,81,251,106,323,56,189,274,14,139,345,207,93,232,167,305,46,119,287,211,34,253,176,329,72,224,151,8,313,98,269,184,51,341,130,291,66,235,159,322,24,198,277,113,350,85,243,161,37,302,124,280,19,216,146,333,61,194,256,102,317,44,170,288,79,226,355,138,11,264,187,314,97,205,53,275,149,326,32,118,238,69,296,163,221,90,347,179,26,250,132,309,57,201,284,114,339,74,228,155,4,273,193,320,41,245,107,299,182,63,212,351,129,16,267,158,306,88,234,47,328,174,101,286,209,35,252,141,315,77,197,343,120,260,58,231,168,292,21,136,304,94,219,65,336,183,242,7,278,153,324,108,202]`},
	{ExpectedCount: 233, Output: `[287,43,196,321,78,154,9,242,113,335,62,219,174,28,303,141,87,264,350,106,183,51,298,127,14,231,92,316,168,74,256,119,342,37,205,83,279,152,6,224,97,331,46,188,273,135,59,310,162,21,245,108,294,71,199,353,32,146,267,90,177,12,238,324,65,158,282,103,49,213,138,305,81,252,19,190,347,116,228,57,314,171,95,260,41,208,133,337,76,185,24,291,149,63,234,101,319,172,8,276,122,344,54,201,86,249,165,30,307,112,223,69,181,328,17,144,271,98,352,47,194,125,286,73,216,4,156,302,89,241,333,58,179,263,109,26,197,340,151,84,229,15,312,137,68,254,187,38,295,120,163,348,93,211,52,281,130,7,237,176,304,114,45,259,182,326,99,22,148,274,61,204,339,128,80,246,11,193,317,155,284,35,221,104,354,167,72,299,143,48,233,91,266,18,189,323,117,55,207,345,132,278,66,244,153,3,309,178,96,225,341,42,160,293,111,77,251,198,29,315,88,236,145,351,53,202,124,269,16,184,308,139,64]`},
	{ExpectedCount: 247, Output: `[284,17,193,342,76,228,119,305,44,261,138,9,317,65,204,351,126,83,273,32,168,294,107,239,51,182,331,94,216,7,149,308,58,253,174,23,337,112,290,136,48,265,199,354,81,222,15,301,157,69,244,103,326,178,39,287,121,346,54,211,98,269,4,163,234,72,313,145,26,257,188,339,116,47,280,201,86,321,152,11,230,297,63,181,349,105,274,36,219,132,306,91,246,171,53,334,20,196,282,125,67,311,159,241,34,176,353,109,263,79,203,18,328,143,295,57,224,97,316,42,185,270,130,6,248,166,344,74,213,299,118,31,258,147,335,88,194,12,281,61,232,154,323,46,207,101,276,24,189,350,135,71,251,162,309,93,217,40,286,114,3,237,179,329,55,202,148,266,84,319,27,173,243,110,352,68,292,155,14,226,340,99,260,43,184,303,122,77,214,333,29,167,288,52,235,104,347,192,8,279,141,64,255,315,87,209,37,150,298,113,272,21,230,175,324,59,197,345,128,45,264,90,312,161,5,249,183,302,75,218,134,355,33,291,106,240,19,170,327,56,205,283,92,146,268,10,223,338,124,49,307,195,80,254]`},
}

func TestUpstreamAstraReference(t *testing.T) {
	wantAstra := []float64{0.9999661480485188, 0.9999999890183678, 0.9999996236029431}
	wantSol := []float64{0.00002741803305652717, 1.0976040351601783e-8, 3.759954729322901e-7}
	for n := 1; n <= 3; n++ {
		got, err := ScoreSamples(astraReference[:n])
		if err != nil {
			t.Fatalf("%d samples: %v", n, err)
		}
		if got.Winner != "gpt-6-astra" || got.Samples != n || len(got.Probabilities) != 16 {
			t.Fatalf("%d samples: unexpected result %+v", n, got)
		}
		for model, want := range map[string]float64{"gpt-6-astra": wantAstra[n-1], "gpt-6-sol": wantSol[n-1]} {
			if !finite(got.Probabilities[model]) || math.Abs(got.Probabilities[model]-want) > math.Max(1e-12, want*1e-6) {
				t.Errorf("%d samples: P(%s) = %.15g, upstream %.15g", n, model, got.Probabilities[model], want)
			}
		}
	}
}

func TestChallengeAndInvalidSamples(t *testing.T) {
	data, err := bankFS.ReadFile("unified_bank.json")
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != "1c2cb74d372f9f0f30d0dabbb7b7a838660d2f769a88d0c8489e4c662e088c21" {
		t.Fatalf("vendored upstream bank changed: %v", err)
	}
	models := Models()
	if len(models) != 8 || !strings.Contains(strings.Join(models, ","), "gpt-6-astra") {
		t.Fatalf("unsupported GPT models: %v", models)
	}
	challenge, err := NewChallenge()
	if err != nil || challenge.System != "" || challenge.ExpectedCount < 292 || challenge.ExpectedCount > 332 || !strings.Contains(challenge.Prompt, "355") {
		t.Fatalf("invalid challenge: %+v, %v", challenge, err)
	}
	if _, err := ScoreSamples([]Sample{astraReference[0], {Output: "refused", ExpectedCount: 300}}); err == nil {
		t.Fatal("partial run should not produce a score")
	}
	if _, err := Score(nil); err == nil {
		t.Fatal("empty run should not produce a score")
	}
	if got := parseNumbers("note 100 200 then 1, 356, 9, 3 items 7"); len(got) != 3 {
		t.Fatalf("longest digit run: %v", got)
	}
	if _, err := ScoreSamples([]Sample{{Output: strings.Repeat("7,", 80), ExpectedCount: 300}}); err == nil {
		t.Fatal("truncated response should not meet the expected-count threshold")
	}
}

func fixtureValue[T any](value any) T {
	result, ok := value.(T)
	if !ok {
		panic(fmt.Sprintf("unexpected fixture type %T", value))
	}
	return result
}

func snapshotData(t *testing.T, edit func(map[string]any)) []byte {
	t.Helper()
	data, err := bankFS.ReadFile("unified_bank.json")
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if edit != nil {
		edit(document)
	}
	data, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestSnapshotActivation(t *testing.T) {
	old := Current()
	t.Cleanup(func() { Activate(old) })
	if old.Version() != Version {
		t.Fatalf("initial version: %s", old.Version())
	}
	want, err := old.ScoreSamples(astraReference)
	if err != nil {
		t.Fatal(err)
	}
	data := snapshotData(t, func(document map[string]any) {
		for _, calibration := range fixtureValue[map[string]any](document["calibration"]) {
			fixtureValue[map[string]any](calibration)["beta"] = 0.5
		}
	})
	next, err := ParseSnapshot(strings.Repeat("a", 40), data)
	if err != nil {
		t.Fatal(err)
	}
	// Neither the input buffer nor a returned model list owns snapshot storage.
	clear(data)
	models := next.Models()
	models[0] = "changed"
	if !reflect.DeepEqual(next.Models(), old.Models()) {
		t.Fatal("model list mutation escaped")
	}
	Activate(next)
	if Current() != next || Current().Version() != strings.Repeat("a", 40) {
		t.Fatal("snapshot was not activated")
	}
	got, err := old.ScoreSamples(astraReference)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("old snapshot changed: %+v, %v", got, err)
	}
	updated, err := ScoreSamples(astraReference)
	if err != nil || reflect.DeepEqual(updated, want) {
		t.Fatalf("package scoring did not switch: %+v, %v", updated, err)
	}
	if direct, err := next.ScoreSamples(astraReference); err != nil || !reflect.DeepEqual(direct, updated) {
		t.Fatalf("package scoring differs from snapshot: %+v, %v", direct, err)
	}
	outputs := []string{astraReference[0].Output}
	if legacy, err := Score(outputs); err != nil {
		t.Fatal(err)
	} else if direct, err := next.ScoreSamples([]Sample{{Output: outputs[0]}}); err != nil || !reflect.DeepEqual(legacy, direct) {
		t.Fatal("legacy Score did not use current snapshot")
	}
	Activate(nil)
	Activate(&Snapshot{})
	if Current() != next {
		t.Fatal("invalid activation replaced current snapshot")
	}
}

func TestParseSnapshotCompatibility(t *testing.T) {
	initial := Current()
	valid := snapshotData(t, nil)
	for _, version := range []string{"", "abc", strings.Repeat("g", 40), strings.Repeat("a", 41), " " + Version} {
		if _, err := ParseSnapshot(version, valid); err == nil {
			t.Errorf("accepted invalid version %q", version)
		}
	}
	for _, data := range [][]byte{nil, []byte("{"), []byte("null"), append(append([]byte(nil), valid...), []byte(" {}")...)} {
		if _, err := ParseSnapshot(Version, data); err == nil {
			t.Error("accepted invalid JSON")
		}
	}
	type mutation struct {
		name string
		edit func(map[string]any)
	}
	tests := []mutation{
		{"schema", func(d map[string]any) { d["schema"] = "other" }},
		{"missing method", func(d map[string]any) { delete(d, "method") }},
		{"queries", func(d map[string]any) { d["recommended_queries"] = 4 }},
		{"minimum", func(d map[string]any) { d["minimum_valid_numbers"] = 1 }},
		{"empty models", func(d map[string]any) { d["models"] = []any{} }},
		{"duplicate ID", func(d map[string]any) {
			models := fixtureValue[[]any](d["models"])
			fixtureValue[map[string]any](models[1])["id"] = fixtureValue[map[string]any](models[0])["id"]
			fixtureValue[[]any](fixtureValue[map[string]any](d["robust"])["model_order"])[1] = fixtureValue[map[string]any](models[0])["id"]
		}},
		{"empty ID", func(d map[string]any) { fixtureValue[map[string]any](fixtureValue[[]any](d["models"])[0])["id"] = "" }},
		{"lost GPT", func(d map[string]any) {
			fixtureValue[map[string]any](fixtureValue[[]any](d["models"])[0])["family"] = "other"
		}},
		{"replaced GPT", func(d map[string]any) {
			fixtureValue[map[string]any](fixtureValue[[]any](d["models"])[0])["id"] = "gpt-replacement"
			fixtureValue[[]any](fixtureValue[map[string]any](d["robust"])["model_order"])[0] = "gpt-replacement"
		}},
		{"missing calibration", func(d map[string]any) { delete(fixtureValue[map[string]any](d["calibration"]), "2") }},
		{"zero beta", func(d map[string]any) {
			fixtureValue[map[string]any](fixtureValue[map[string]any](d["calibration"])["1"])["beta"] = 0
		}},
		{"negative beta", func(d map[string]any) {
			fixtureValue[map[string]any](fixtureValue[map[string]any](d["calibration"])["1"])["beta"] = -1
		}},
	}
	var document map[string]any
	if err := json.Unmarshal(valid, &document); err != nil {
		t.Fatal(err)
	}
	for field := range fixtureValue[map[string]any](document["method"]) {
		tests = append(tests, mutation{"method " + field, func(d map[string]any) { fixtureValue[map[string]any](d["method"])[field] = nil }})
	}
	tests = append(tests, mutation{"unknown method", func(d map[string]any) { fixtureValue[map[string]any](d["method"])["future_rule"] = "unsupported" }})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ParseSnapshot(Version, snapshotData(t, test.edit)); err == nil {
				t.Fatal("accepted incompatible bank")
			}
		})
	}
	robustTests := []mutation{
		{"order", func(r map[string]any) { fixtureValue[[]any](r["model_order"])[0] = "other" }},
		{"order length", func(r map[string]any) { r["model_order"] = []any{} }},
		{"not ready", func(r map[string]any) { r["robust_ready"] = false }},
		{"environments", func(r map[string]any) { r["complete_environments"] = []any{} }},
		{"environment order", func(r map[string]any) {
			fixtureValue[[]any](fixtureValue[map[string]any](r["hellinger"])["nuisance_environments"])[0] = "other"
		}},
		{"duplicate environment", func(r map[string]any) {
			fixtureValue[[]any](r["complete_environments"])[1] = fixtureValue[[]any](r["complete_environments"])[0]
			fixtureValue[[]any](fixtureValue[map[string]any](r["hellinger"])["nuisance_environments"])[1] = fixtureValue[[]any](r["complete_environments"])[0]
		}},
		{"weight", func(r map[string]any) { fixtureValue[map[string]any](r["ordered_blocks"])["weight"] = 0.5 }},
		{"feature rule", func(r map[string]any) { fixtureValue[map[string]any](r["ordered_blocks"])["feature"] = "other" }},
		{"missing environment centroids", func(r map[string]any) {
			delete(fixtureValue[map[string]any](r["ordered_blocks"]), "environment_centroids")
		}},
		{"environment count", func(r map[string]any) {
			fixtureValue[map[string]any](r["ordered_blocks"])["environment_centroids"] = []any{}
		}},
		{"environment models", func(r map[string]any) {
			fixtureValue[[]any](fixtureValue[map[string]any](r["ordered_blocks"])["environment_centroids"])[0] = []any{}
		}},
		{"environment dimension", func(r map[string]any) {
			fixtureValue[[]any](fixtureValue[[]any](fixtureValue[map[string]any](r["ordered_blocks"])["environment_centroids"])[0])[0] = []any{1.0}
		}},
	}
	for _, name := range []string{"hellinger", "ordered_blocks"} {
		for _, field := range []string{"feature_mean", "feature_scale", "nuisance_basis", "centroids"} {
			robustTests = append(robustTests, mutation{name + " " + field, func(r map[string]any) { fixtureValue[map[string]any](r[name])[field] = []any{} }})
		}
		robustTests = append(robustTests,
			mutation{name + " centroid dimension", func(r map[string]any) {
				fixtureValue[[]any](fixtureValue[map[string]any](r[name])["centroids"])[0] = []any{1.0}
			}},
			mutation{name + " basis dimension", func(r map[string]any) {
				fixtureValue[[]any](fixtureValue[map[string]any](r[name])["nuisance_basis"])[0] = []any{1.0}
			}},
			mutation{name + " rank", func(r map[string]any) { fixtureValue[map[string]any](r[name])["nuisance_rank"] = 100 }},
			mutation{name + " missing rank", func(r map[string]any) { delete(fixtureValue[map[string]any](r[name]), "nuisance_rank") }})
		for _, value := range []any{0.0, -1.0, nil} {
			robustTests = append(robustTests, mutation{fmt.Sprintf("%s scale %v", name, value), func(r map[string]any) {
				fixtureValue[[]any](fixtureValue[map[string]any](r[name])["feature_scale"])[0] = value
			}})
		}
		robustTests = append(robustTests, mutation{name + " null mean", func(r map[string]any) {
			fixtureValue[[]any](fixtureValue[map[string]any](r[name])["feature_mean"])[0] = nil
		}})
	}
	for _, test := range robustTests {
		t.Run(test.name, func(t *testing.T) {
			data := snapshotData(t, func(d map[string]any) { test.edit(fixtureValue[map[string]any](d["robust"])) })
			if _, err := ParseSnapshot(Version, data); err == nil {
				t.Fatal("accepted incompatible robust data")
			}
		})
	}
	for _, field := range []string{"beta", "weight", "feature_mean"} {
		data := snapshotData(t, func(d map[string]any) {
			switch field {
			case "beta":
				fixtureValue[map[string]any](fixtureValue[map[string]any](d["calibration"])["1"])[field] = "invalid-number"
			case "weight":
				fixtureValue[map[string]any](fixtureValue[map[string]any](d["robust"])["ordered_blocks"])[field] = "invalid-number"
			default:
				fixtureValue[[]any](fixtureValue[map[string]any](fixtureValue[map[string]any](d["robust"])["hellinger"])[field])[0] = "invalid-number"
			}
		})
		for _, invalid := range []string{"1e999", "NaN", "Infinity"} {
			if _, err := ParseSnapshot(Version, []byte(strings.Replace(string(data), `"invalid-number"`, invalid, 1))); err == nil {
				t.Errorf("accepted %s %s", field, invalid)
			}
		}
	}
	if Current() != initial {
		t.Fatal("parsing changed the active snapshot")
	}
}

func TestSnapshotAllowsNewModels(t *testing.T) {
	data := snapshotData(t, func(d map[string]any) {
		models := fixtureValue[[]any](d["models"])
		d["models"] = append(models, map[string]any{"id": "gpt-future", "family": "gpt"})
		robust := fixtureValue[map[string]any](d["robust"])
		robust["model_order"] = append(fixtureValue[[]any](robust["model_order"]), "gpt-future")
		for _, name := range []string{"hellinger", "ordered_blocks"} {
			artifact := fixtureValue[map[string]any](robust[name])
			centroids := fixtureValue[[]any](artifact["centroids"])
			artifact["centroids"] = append(centroids, centroids[0])
			if name == "ordered_blocks" {
				for i, environment := range fixtureValue[[]any](artifact["environment_centroids"]) {
					rows := fixtureValue[[]any](environment)
					fixtureValue[[]any](artifact["environment_centroids"])[i] = append(rows, rows[0])
				}
			}
		}
	})
	next, err := ParseSnapshot(strings.Repeat("b", 40), data)
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Models()) != 9 || next.Models()[8] != "gpt-future" {
		t.Fatalf("new model missing: %v", next.Models())
	}
	got, err := next.ScoreSamples(astraReference)
	if err != nil || len(got.Probabilities) != 17 {
		t.Fatalf("new model not scored: %+v, %v", got, err)
	}
}

func TestConcurrentSnapshotScoring(t *testing.T) {
	old := Current()
	t.Cleanup(func() { Activate(old) })
	next, err := ParseSnapshot(strings.Repeat("c", 40), snapshotData(t, func(d map[string]any) {
		robust := fixtureValue[map[string]any](d["robust"])
		// Permute both scoring branches, keeping IDs aligned with every matrix.
		models := fixtureValue[[]any](d["models"])
		models[0], models[5] = models[5], models[0]
		order := fixtureValue[[]any](robust["model_order"])
		order[0], order[5] = order[5], order[0]
		for _, name := range []string{"hellinger", "ordered_blocks"} {
			artifact := fixtureValue[map[string]any](robust[name])
			rows := fixtureValue[[]any](artifact["centroids"])
			rows[0], rows[5] = rows[5], rows[0]
			if name == "ordered_blocks" {
				for _, environment := range fixtureValue[[]any](artifact["environment_centroids"]) {
					rows := fixtureValue[[]any](environment)
					rows[0], rows[5] = rows[5], rows[0]
				}
			}
		}
		for _, calibration := range fixtureValue[map[string]any](d["calibration"]) {
			fixtureValue[map[string]any](calibration)["beta"] = 0.5
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	wantOld, err := old.ScoreSamples(astraReference)
	if err != nil {
		t.Fatal(err)
	}
	wantNew, err := next.ScoreSamples(astraReference)
	if err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	var writer, readers sync.WaitGroup
	writer.Go(func() {
		for {
			select {
			case <-stop:
				return
			default:
				Activate(next)
				runtime.Gosched()
				Activate(old)
				runtime.Gosched()
			}
		}
	})
	for range 4 {
		readers.Go(func() {
			for range 100 {
				got, err := ScoreSamples(astraReference)
				if err != nil || (!reflect.DeepEqual(got, wantOld) && !reflect.DeepEqual(got, wantNew)) {
					t.Errorf("mixed scoring banks: %+v, %v", got, err)
					return
				}
				if got, err := old.ScoreSamples(astraReference); err != nil || !reflect.DeepEqual(got, wantOld) {
					t.Errorf("captured snapshot changed: %+v, %v", got, err)
					return
				}
			}
		})
	}
	readers.Wait()
	close(stop)
	writer.Wait()
}

func TestSnapshotRejectsNonFiniteScore(t *testing.T) {
	data := snapshotData(t, func(d map[string]any) {
		fixtureValue[map[string]any](fixtureValue[map[string]any](d["calibration"])["1"])["beta"] = math.MaxFloat64
	})
	if _, err := ParseSnapshot(Version, data); err == nil {
		t.Fatal("extreme calibration must be rejected during parsing")
	}
	var bank fingerprintBank
	if err := json.Unmarshal(data, &bank); err != nil {
		t.Fatal(err)
	}
	next := &Snapshot{version: Version, bank: bank}
	if _, err := next.ScoreSamples(astraReference[:1]); err == nil {
		t.Fatal("calibration overflow must not produce NaN probabilities")
	}
}

func TestFingerprintNumericalRanges(t *testing.T) {
	for _, name := range []string{"hellinger", "ordered_blocks"} {
		for _, test := range []struct {
			name  string
			field string
			value float64
		}{
			{"negative mean", "feature_mean", -0.01},
			{"large mean", "feature_mean", 1.01},
			{"small scale", "feature_scale", 1e-13},
			{"large scale", "feature_scale", 1.01},
		} {
			t.Run(name+"/"+test.name, func(t *testing.T) {
				data := snapshotData(t, func(d map[string]any) {
					fixtureValue[[]any](fixtureValue[map[string]any](fixtureValue[map[string]any](d["robust"])[name])[test.field])[0] = test.value
				})
				if _, err := ParseSnapshot(Version, data); err == nil {
					t.Fatal("accepted number outside upstream feature range")
				}
			})
		}
		for _, field := range []string{"nuisance_basis", "centroids"} {
			t.Run(name+"/"+field+" norm", func(t *testing.T) {
				data := snapshotData(t, func(d map[string]any) {
					row := fixtureValue[[]any](fixtureValue[[]any](fixtureValue[map[string]any](fixtureValue[map[string]any](d["robust"])[name])[field])[0])
					row[0], row[1] = 1.0, 1.0
				})
				if _, err := ParseSnapshot(Version, data); err == nil {
					t.Fatal("accepted excessive norm with individually valid components")
				}
			})
		}
	}
	for _, beta := range []float64{0.01, 13, 1e200} {
		data := snapshotData(t, func(d map[string]any) {
			fixtureValue[map[string]any](fixtureValue[map[string]any](d["calibration"])["1"])["beta"] = beta
		})
		if _, err := ParseSnapshot(Version, data); err == nil {
			t.Errorf("accepted beta outside upstream search range: %g", beta)
		}
	}
	for _, bounds := range []struct {
		mean, scale, beta float64
	}{{0, 1e-12, 0.05}, {1, 1, 12}} {
		data := snapshotData(t, func(d map[string]any) {
			for _, name := range []string{"hellinger", "ordered_blocks"} {
				artifact := fixtureValue[map[string]any](fixtureValue[map[string]any](d["robust"])[name])
				for i := range fixtureValue[[]any](artifact["feature_mean"]) {
					fixtureValue[[]any](artifact["feature_mean"])[i] = bounds.mean
					fixtureValue[[]any](artifact["feature_scale"])[i] = bounds.scale
				}
			}
			for _, calibration := range fixtureValue[map[string]any](d["calibration"]) {
				fixtureValue[map[string]any](calibration)["beta"] = bounds.beta
			}
		})
		next, err := ParseSnapshot(Version, data)
		if err != nil {
			t.Fatalf("rejected valid numeric boundary %+v: %v", bounds, err)
		}
		for n := 1; n <= 3; n++ {
			got, err := next.ScoreSamples(astraReference[:n])
			if err != nil {
				t.Fatalf("numeric boundary %+v, %d samples: %v", bounds, n, err)
			}
			var total float64
			for _, probability := range got.Probabilities {
				if !finite(probability) || probability < 0 || probability > 1 {
					t.Fatalf("invalid boundary probability: %g", probability)
				}
				total += probability
			}
			if math.Abs(total-1) > 1e-12 {
				t.Fatalf("probabilities sum to %g", total)
			}
		}
	}
}

func TestExtremeFingerprintNumbers(t *testing.T) {
	initial := Current()
	for _, name := range []string{"hellinger", "ordered_blocks"} {
		for _, test := range []struct {
			name string
			edit func(map[string]any)
		}{
			{"mean", func(a map[string]any) { fixtureValue[[]any](a["feature_mean"])[0] = 1e200 }},
			{"tiny scale", func(a map[string]any) { fixtureValue[[]any](a["feature_scale"])[0] = 1e-200 }},
			{"basis", func(a map[string]any) { fixtureValue[[]any](fixtureValue[[]any](a["nuisance_basis"])[0])[0] = 1e200 }},
			{"centroid", func(a map[string]any) { fixtureValue[[]any](fixtureValue[[]any](a["centroids"])[0])[0] = 1e200 }},
		} {
			t.Run(name+"/"+test.name, func(t *testing.T) {
				data := snapshotData(t, func(d map[string]any) {
					test.edit(fixtureValue[map[string]any](fixtureValue[map[string]any](d["robust"])[name]))
				})
				if _, err := ParseSnapshot(Version, data); err == nil {
					t.Error("extreme finite number must be rejected during parsing")
				}
				// Bypass parsing to exercise the scoring guard on the real reference outputs.
				var bank fingerprintBank
				if err := json.Unmarshal(data, &bank); err != nil {
					t.Fatal(err)
				}
				invalid := &Snapshot{version: Version, bank: bank}
				if got, err := invalid.ScoreSamples(astraReference); err == nil {
					t.Fatalf("overflow silently produced winner %s: %+v", got.Winner, got.Probabilities)
				} else if !reflect.DeepEqual(got, Result{}) {
					t.Fatalf("failed scoring returned a partial conclusion: %+v", got)
				}
			})
		}
	}
	t.Run("environment centroid", func(t *testing.T) {
		data := snapshotData(t, func(d map[string]any) {
			fixtureValue[[]any](fixtureValue[[]any](fixtureValue[[]any](fixtureValue[map[string]any](fixtureValue[map[string]any](d["robust"])["ordered_blocks"])["environment_centroids"])[0])[0])[0] = 1e200
		})
		if _, err := ParseSnapshot(Version, data); err == nil {
			t.Error("extreme environment centroid must be rejected")
		}
		var bank fingerprintBank
		if err := json.Unmarshal(data, &bank); err != nil {
			t.Fatal(err)
		}
		if got, err := (&Snapshot{version: Version, bank: bank}).ScoreSamples(astraReference); err == nil {
			t.Fatalf("environment overflow silently produced winner %s", got.Winner)
		}
	})
	if Current() != initial {
		t.Fatal("failed parsing changed the current snapshot")
	}
}
