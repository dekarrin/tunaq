package ictiobus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_CompleteRun(t *testing.T) {
	assert := assert.New(t)

	actual := ReadFishiMdFile("fishi.md")

	assert.NoError(actual)
	nactual := false
	assert.True(nactual)
}

func Test_GetFishiFromMarkdown(t *testing.T) {
	testCases := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name: "fishi and text",
			input: "Test block\n" +
				"only include the fishi block\n" +
				"```fishi\n" +
				"%%tokens\n" +
				"\n" +
				"%token test\n" +
				"```\n",
			expect: "%%tokens\n" +
				"\n" +
				"%token test\n",
		},
		{
			name: "two fishi blocks",
			input: "Test block\n" +
				"only include the fishi blocks\n" +
				"```fishi\n" +
				"%%tokens\n" +
				"\n" +
				"%token test\n" +
				"```\n" +
				"some more text\n" +
				"```fishi\n" +
				"\n" +
				"%token 7\n" +
				"%%actions\n" +
				"\n" +
				"%action go\n" +
				"```\n" +
				"other text\n",
			expect: "%%tokens\n" +
				"\n" +
				"%token test\n" +
				"\n" +
				"%token 7\n" +
				"%%actions\n" +
				"\n" +
				"%action go\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			actual := GetFishiFromMarkdown([]byte(tc.input))

			assert.Equal(tc.expect, string(actual))
		})
	}
}
