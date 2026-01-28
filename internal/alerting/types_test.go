package alerting

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAlertSeverity_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		severity AlertSeverity
		want     bool
	}{
		{name: "info is valid", severity: SeverityInfo, want: true},
		{name: "warning is valid", severity: SeverityWarning, want: true},
		{name: "critical is valid", severity: SeverityCritical, want: true},
		{name: "empty is invalid", severity: "", want: false},
		{name: "unknown is invalid", severity: "unknown", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.severity.IsValid(); got != tt.want {
				t.Errorf("AlertSeverity.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAlertSeverity_StringValues(t *testing.T) {
	tests := []struct {
		severity AlertSeverity
		want     string
	}{
		{SeverityInfo, "info"},
		{SeverityWarning, "warning"},
		{SeverityCritical, "critical"},
	}

	for _, tt := range tests {
		if string(tt.severity) != tt.want {
			t.Errorf("AlertSeverity = %q, want %q", tt.severity, tt.want)
		}
	}
}

func TestAlertStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status AlertStatus
		want   bool
	}{
		{name: "firing is valid", status: StatusFiring, want: true},
		{name: "resolved is valid", status: StatusResolved, want: true},
		{name: "empty is invalid", status: "", want: false},
		{name: "unknown is invalid", status: "unknown", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("AlertStatus.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAlertStatus_StringValues(t *testing.T) {
	tests := []struct {
		status AlertStatus
		want   string
	}{
		{StatusFiring, "firing"},
		{StatusResolved, "resolved"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("AlertStatus = %q, want %q", tt.status, tt.want)
		}
	}
}

func TestOperator_IsValid(t *testing.T) {
	tests := []struct {
		name string
		op   Operator
		want bool
	}{
		{name: "gt is valid", op: OpGreaterThan, want: true},
		{name: "lt is valid", op: OpLessThan, want: true},
		{name: "eq is valid", op: OpEqual, want: true},
		{name: "gte is valid", op: OpGreaterThanEqual, want: true},
		{name: "lte is valid", op: OpLessThanEqual, want: true},
		{name: "empty is invalid", op: "", want: false},
		{name: "unknown is invalid", op: "neq", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op.IsValid(); got != tt.want {
				t.Errorf("Operator.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOperator_Evaluate(t *testing.T) {
	tests := []struct {
		name      string
		op        Operator
		value     float64
		threshold float64
		want      bool
	}{
		// Greater than
		{name: "gt: 10 > 5", op: OpGreaterThan, value: 10, threshold: 5, want: true},
		{name: "gt: 5 > 10", op: OpGreaterThan, value: 5, threshold: 10, want: false},
		{name: "gt: 5 > 5", op: OpGreaterThan, value: 5, threshold: 5, want: false},

		// Less than
		{name: "lt: 5 < 10", op: OpLessThan, value: 5, threshold: 10, want: true},
		{name: "lt: 10 < 5", op: OpLessThan, value: 10, threshold: 5, want: false},
		{name: "lt: 5 < 5", op: OpLessThan, value: 5, threshold: 5, want: false},

		// Equal
		{name: "eq: 5 == 5", op: OpEqual, value: 5, threshold: 5, want: true},
		{name: "eq: 5 == 10", op: OpEqual, value: 5, threshold: 10, want: false},

		// Greater than or equal
		{name: "gte: 10 >= 5", op: OpGreaterThanEqual, value: 10, threshold: 5, want: true},
		{name: "gte: 5 >= 5", op: OpGreaterThanEqual, value: 5, threshold: 5, want: true},
		{name: "gte: 4 >= 5", op: OpGreaterThanEqual, value: 4, threshold: 5, want: false},

		// Less than or equal
		{name: "lte: 5 <= 10", op: OpLessThanEqual, value: 5, threshold: 10, want: true},
		{name: "lte: 5 <= 5", op: OpLessThanEqual, value: 5, threshold: 5, want: true},
		{name: "lte: 6 <= 5", op: OpLessThanEqual, value: 6, threshold: 5, want: false},

		// Edge cases
		{name: "gt: 0 > 0", op: OpGreaterThan, value: 0, threshold: 0, want: false},
		{name: "lt: negative values", op: OpLessThan, value: -10, threshold: -5, want: true},
		{name: "eq: floats", op: OpEqual, value: 3.14159, threshold: 3.14159, want: true},
		{name: "gte: very small diff", op: OpGreaterThanEqual, value: 1.0000001, threshold: 1.0, want: true},

		// Invalid operator
		{name: "invalid operator", op: "invalid", value: 5, threshold: 5, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.op.Evaluate(tt.value, tt.threshold); got != tt.want {
				t.Errorf("Operator.Evaluate(%v, %v) = %v, want %v", tt.value, tt.threshold, got, tt.want)
			}
		})
	}
}

func TestOperator_String(t *testing.T) {
	tests := []struct {
		op   Operator
		want string
	}{
		{OpGreaterThan, ">"},
		{OpLessThan, "<"},
		{OpEqual, "=="},
		{OpGreaterThanEqual, ">="},
		{OpLessThanEqual, "<="},
		{Operator("unknown"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.op), func(t *testing.T) {
			if got := tt.op.String(); got != tt.want {
				t.Errorf("Operator.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestChannelType_IsValid(t *testing.T) {
	tests := []struct {
		name        string
		channelType ChannelType
		want        bool
	}{
		{name: "slack is valid", channelType: ChannelSlack, want: true},
		{name: "email is valid", channelType: ChannelEmail, want: true},
		{name: "webhook is valid", channelType: ChannelWebhook, want: true},
		{name: "pagerduty is valid", channelType: ChannelPagerDuty, want: true},
		{name: "empty is invalid", channelType: "", want: false},
		{name: "unknown is invalid", channelType: "sms", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.channelType.IsValid(); got != tt.want {
				t.Errorf("ChannelType.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateFingerprint(t *testing.T) {
	ruleID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	tests := []struct {
		name   string
		ruleID uuid.UUID
		labels map[string]string
	}{
		{
			name:   "empty labels",
			ruleID: ruleID,
			labels: map[string]string{},
		},
		{
			name:   "single label",
			ruleID: ruleID,
			labels: map[string]string{"source": "db1"},
		},
		{
			name:   "multiple labels",
			ruleID: ruleID,
			labels: map[string]string{"source": "db1", "table": "users"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp1 := GenerateFingerprint(tt.ruleID, tt.labels)
			fp2 := GenerateFingerprint(tt.ruleID, tt.labels)

			// Fingerprint should be consistent
			if fp1 != fp2 {
				t.Errorf("GenerateFingerprint() not consistent: %q != %q", fp1, fp2)
			}

			// Fingerprint should be a valid hex string (SHA256 = 64 chars)
			if len(fp1) != 64 {
				t.Errorf("GenerateFingerprint() length = %d, want 64", len(fp1))
			}
		})
	}
}

func TestGenerateFingerprint_DifferentInputs(t *testing.T) {
	ruleID1 := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	ruleID2 := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")

	labels1 := map[string]string{"source": "db1"}
	labels2 := map[string]string{"source": "db2"}

	fp1 := GenerateFingerprint(ruleID1, labels1)
	fp2 := GenerateFingerprint(ruleID2, labels1)
	fp3 := GenerateFingerprint(ruleID1, labels2)

	// Different rule IDs should produce different fingerprints
	if fp1 == fp2 {
		t.Error("Different rule IDs should produce different fingerprints")
	}

	// Different labels should produce different fingerprints
	if fp1 == fp3 {
		t.Error("Different labels should produce different fingerprints")
	}
}

func TestGenerateFingerprint_LabelOrderIndependence(t *testing.T) {
	ruleID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	// Labels added in different order
	labels1 := map[string]string{"a": "1", "b": "2", "c": "3"}
	labels2 := map[string]string{"c": "3", "a": "1", "b": "2"}
	labels3 := map[string]string{"b": "2", "c": "3", "a": "1"}

	fp1 := GenerateFingerprint(ruleID, labels1)
	fp2 := GenerateFingerprint(ruleID, labels2)
	fp3 := GenerateFingerprint(ruleID, labels3)

	// Fingerprints should be identical regardless of label order
	if fp1 != fp2 || fp2 != fp3 {
		t.Errorf("Fingerprints should be identical regardless of label order: %q, %q, %q", fp1, fp2, fp3)
	}
}

func TestAlertSilence_IsActive(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		silence AlertSilence
		want    bool
	}{
		{
			name: "active silence",
			silence: AlertSilence{
				StartsAt: now.Add(-1 * time.Hour),
				EndsAt:   now.Add(1 * time.Hour),
			},
			want: true,
		},
		{
			name: "expired silence",
			silence: AlertSilence{
				StartsAt: now.Add(-2 * time.Hour),
				EndsAt:   now.Add(-1 * time.Hour),
			},
			want: false,
		},
		{
			name: "future silence",
			silence: AlertSilence{
				StartsAt: now.Add(1 * time.Hour),
				EndsAt:   now.Add(2 * time.Hour),
			},
			want: false,
		},
		{
			name: "silence about to start (boundary)",
			silence: AlertSilence{
				StartsAt: now.Add(1 * time.Second),
				EndsAt:   now.Add(1 * time.Hour),
			},
			want: false, // Not yet started
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.silence.IsActive(); got != tt.want {
				t.Errorf("AlertSilence.IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAlertSilence_Matches(t *testing.T) {
	tests := []struct {
		name     string
		matchers map[string]string
		labels   map[string]string
		want     bool
	}{
		{
			name:     "empty matchers match everything",
			matchers: map[string]string{},
			labels:   map[string]string{"source": "db1"},
			want:     true,
		},
		{
			name:     "exact match",
			matchers: map[string]string{"source": "db1"},
			labels:   map[string]string{"source": "db1"},
			want:     true,
		},
		{
			name:     "partial match with extra labels",
			matchers: map[string]string{"source": "db1"},
			labels:   map[string]string{"source": "db1", "table": "users"},
			want:     true,
		},
		{
			name:     "no match - different value",
			matchers: map[string]string{"source": "db1"},
			labels:   map[string]string{"source": "db2"},
			want:     false,
		},
		{
			name:     "no match - missing label",
			matchers: map[string]string{"source": "db1", "env": "prod"},
			labels:   map[string]string{"source": "db1"},
			want:     false,
		},
		{
			name:     "multiple matchers all match",
			matchers: map[string]string{"source": "db1", "env": "prod"},
			labels:   map[string]string{"source": "db1", "env": "prod", "table": "users"},
			want:     true,
		},
		{
			name:     "multiple matchers one fails",
			matchers: map[string]string{"source": "db1", "env": "prod"},
			labels:   map[string]string{"source": "db1", "env": "staging"},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			silence := &AlertSilence{Matchers: tt.matchers}
			if got := silence.Matches(tt.labels); got != tt.want {
				t.Errorf("AlertSilence.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}
