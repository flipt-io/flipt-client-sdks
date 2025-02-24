package evaluation_test

import (
	"context"
	"os"
	"testing"

	flipt "go.flipt.io/flipt-client"
)

var (
	benchClient *flipt.EvaluationClient
)

func init() {
	opts := []flipt.ClientOption{}

	if os.Getenv("FLIPT_URL") != "" {
		opts = append(opts, flipt.WithURL(os.Getenv("FLIPT_URL")))
	}

	if os.Getenv("FLIPT_AUTH_TOKEN") != "" {
		opts = append(opts, flipt.WithClientTokenAuthentication(os.Getenv("FLIPT_AUTH_TOKEN")))
	}

	var err error
	benchClient, err = flipt.NewEvaluationClient(
		context.TODO(),
		opts...,
	)
	if err != nil {
		panic(err)
	}
}

func BenchmarkVariantEvaluation(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := benchClient.EvaluateVariant(ctx, "flag1", "someentity", map[string]string{
			"fizz": "buzz",
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBooleanEvaluation(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := benchClient.EvaluateBoolean(ctx, "flag_boolean", "someentity", map[string]string{
			"fizz": "buzz",
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBatchEvaluation(b *testing.B) {
	ctx := context.Background()
	requests := []*flipt.EvaluationRequest{
		{
			FlagKey:  "flag1",
			EntityId: "someentity",
			Context: map[string]string{
				"fizz": "buzz",
			},
		},
		{
			FlagKey:  "flag_boolean",
			EntityId: "someentity",
			Context: map[string]string{
				"fizz": "buzz",
			},
		},
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := benchClient.EvaluateBatch(ctx, requests)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkListFlags(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := benchClient.ListFlags(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
