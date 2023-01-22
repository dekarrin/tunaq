package tunascript

/*
func Test_Parse(t *testing.T) {
	testCases := []struct {
		name      string
		input     []token
		expect    AST
		expectErr bool
	}{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			inputTokenStream := tokenStream{
				tokens: tc.input,
			}

			actualAST, err := Parse(inputTokenStream)
			if tc.expectErr {
				assert.Error(err)
			}
			assert.Nil(err)

			assert.Equal(strings.Join(expectStrings, " "), strings.Join(actualStrings, " "))
		})
	}
}*/

func sRef(s string) *string {
	return &s
}

func iRef(i int) *int {
	return &i
}

func bRef(b bool) *bool {
	return &b
}

type wi struct {
	inInven bool
	move    bool
	output  bool
}

func (w wi) InInventory(label string) bool {
	return w.inInven
}

func (w wi) Move(label string, dest string) bool {
	return w.move
}

func (w wi) Output(str string) bool {
	return w.output
}

func worldInterFixture() WorldInterface {
	return wi{}
}
