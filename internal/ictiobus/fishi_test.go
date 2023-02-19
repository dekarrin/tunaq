package ictiobus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			actual := GetFishiFromMarkdown([]byte(tc.input))

			assert.Equal(tc.expect, string(actual))
		})
	}
}
