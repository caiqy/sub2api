package service

import (
	"context"
	"slices"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/modeltrace"
	"github.com/stretchr/testify/require"
)

func TestModelTraceRemovedAcceptedModelsRemainRemovable(t *testing.T) {
	ctx := context.Background()
	settings := &SettingService{settingRepo: &modelTraceSettingsStub{raw: `{"accepted_models":["removed-model"]}`}}
	cfg, err := settings.GetModelTraceSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"removed-model"}, cfg.AcceptedModels)
	require.Error(t, settings.SetModelTraceSettings(ctx, cfg))
	cfg.AcceptedModels = []string{}
	require.NoError(t, settings.SetModelTraceSettings(ctx, cfg))
	got, err := settings.GetModelTraceSettings(ctx)
	require.NoError(t, err)
	require.Empty(t, got.AcceptedModels)
}

func TestModelTraceAcceptedModelsPreserveAttributionAndUseTaskSnapshot(t *testing.T) {
	// First upstream Astra reference sample, shared with pkg/modeltrace's scoring regression.
	output := `[247,83,319,146,28,204,337,112,59,271,164,352,91,223,17,308,135,76,289,192,43,258,121,334,68,215,301,154,9,237,178,325,54,109,283,196,31,342,87,261,143,22,310,185,73,229,348,116,294,48,203,157,331,95,266,12,181,316,64,240,127,354,39,218,172,298,81,251,106,323,56,189,274,14,139,345,207,93,232,167,305,46,119,287,211,34,253,176,329,72,224,151,8,313,98,269,184,51,341,130,291,66,235,159,322,24,198,277,113,350,85,243,161,37,302,124,280,19,216,146,333,61,194,256,102,317,44,170,288,79,226,355,138,11,264,187,314,97,205,53,275,149,326,32,118,238,69,296,163,221,90,347,179,26,250,132,309,57,201,284,114,339,74,228,155,4,273,193,320,41,245,107,299,182,63,212,351,129,16,267,158,306,88,234,47,328,174,101,286,209,35,252,141,315,77,197,343,120,260,58,231,168,292,21,136,304,94,219,65,336,183,242,7,278,153,324,108,202]`
	expected, err := modeltrace.Score([]string{output})
	require.NoError(t, err)
	models := modeltrace.Models()
	require.Contains(t, models, expected.Winner)
	other := models[0]
	if other == expected.Winner {
		other = models[1]
	}
	for _, source := range []string{"manual", "auto"} {
		for _, tc := range []struct {
			name, target, result string
			accepted             []string
		}{
			{"target always accepted", expected.Winner, "normal", []string{other}},
			{"additional accepted", other, "normal", []string{expected.Winner}},
			{"unlisted winner", other, "degraded", []string{other}},
			{"legacy empty list", other, "degraded", nil},
		} {
			t.Run(source+"/"+tc.name, func(t *testing.T) {
				ctx := context.Background()
				settings := &SettingService{settingRepo: &modelTraceSettingsStub{}}
				cfg, err := settings.GetModelTraceSettings(ctx)
				require.NoError(t, err)
				// Deliberately opposite to the task: execution must use its persisted snapshot.
				cfg.Enabled = true
				if !slices.Contains(tc.accepted, expected.Winner) {
					cfg.AcceptedModels = []string{expected.Winner}
				}
				require.NoError(t, settings.SetModelTraceSettings(ctx, cfg))
				account := &Account{ID: 1, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true,
					Extra: map[string]any{ModelTraceEnabledExtraKey: true, ModelTraceModelExtraKey: tc.target}}
				tasks := &modelTraceTasksStub{}
				probe := &modelTraceProbeStub{account: account, target: tc.target, output: output}
				s := &ModelTraceService{ctx: ctx, repo: tasks, accounts: &modelTraceAccountsStub{account: account}, settings: settings, prober: probe}
				task := &ModelTraceTask{ID: 1, AccountID: 1, Source: source, Model: tc.target, TargetModel: tc.target,
					ModelOverride: source == "auto", Rounds: 1, AcceptedModels: tc.accepted}
				s.execute(ctx, "owner", task)
				require.Equal(t, "completed", tasks.finished.Status, tasks.finished.Error)
				require.Equal(t, tc.result, tasks.finished.Result)
				require.Equal(t, expected.Winner, tasks.finished.Winner)
				require.Equal(t, expected.Probabilities, tasks.finished.Probabilities)
			})
		}
	}
}
