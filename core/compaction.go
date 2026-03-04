package core

const tokensPerChar = 0.25

func EstimateTokens(msgs []Message) int {
	total := 0
	for _, m := range msgs {
		total += int(float64(len(m.Content)) * tokensPerChar)
		total += 4
	}
	return total
}

func NeedsCompaction(msgs []Message, contextWindow int) bool {
	return EstimateTokens(msgs) > int(float64(contextWindow)*0.75)
}
