package syntax

import "fmt"

type BinaryOperation int

const (
	Equal BinaryOperation = iota
	NotEqual
	LessThan
	LessThanEqual
	GreaterThan
	GreaterThanEqual
	Add
	Subtract
	Divide
	Multiply
	LogicalAnd
	LogicalOr
)

// Symbol returns the Tunascript operator that corresponds to the operation.
func (op BinaryOperation) Symbol() string {
	switch op {
	case Equal:
		return "="
	case NotEqual:
		return "!="
	case LessThan:
		return "<"
	case LessThanEqual:
		return "<="
	case GreaterThan:
		return ">"
	case GreaterThanEqual:
		return ">="
	case Add:
		return "+"
	case Subtract:
		return "-"
	case Divide:
		return "/"
	case Multiply:
		return "*"
	case LogicalAnd:
		return "&&"
	case LogicalOr:
		return "||"
	default:
		panic(fmt.Sprintf("unknown binary operation: %d", op))
	}
}

func (op BinaryOperation) String() string {
	switch op {
	case Equal:
		return "EQUALITY"
	case NotEqual:
		return "NON_EQUALITY"
	case LessThan:
		return "LESS_THAN"
	case LessThanEqual:
		return "LESS_THAN_EQUALITY"
	case GreaterThan:
		return "GREATER_THAN"
	case GreaterThanEqual:
		return "GREATER_THAN_EQUALITY"
	case Add:
		return "ADDITION"
	case Subtract:
		return "SUBTRACTION"
	case Divide:
		return "DIVISION"
	case Multiply:
		return "MULTIPLICATION"
	case LogicalAnd:
		return "AND"
	case LogicalOr:
		return "OR"
	default:
		panic(fmt.Sprintf("unknown binary operation: %d", op))
	}
}

type UnaryOperation int

const (
	Negate UnaryOperation = iota
	LogicalNot
)

// Symbol returns the Tunascript operator that corresponds to the operation.
func (op UnaryOperation) Symbol() string {
	switch op {
	case Negate:
		return "-"
	case LogicalNot:
		return "!"
	default:
		panic(fmt.Sprintf("unknown unary operation: %d", op))
	}
}

// String returns the Tunascript operator that corresponds to the operation.
func (op UnaryOperation) String() string {
	switch op {
	case Negate:
		return "NEGATION"
	case LogicalNot:
		return "NOT"
	default:
		panic(fmt.Sprintf("unknown unary operation: %d", op))
	}
}

type AssignmentOperation int

const (
	Set AssignmentOperation = iota
	Increment
	Decrement
	IncrementBy
	DecrementBy
)

// Symbol returns the Tunascript operator that corresponds to the operation.
func (op AssignmentOperation) Symbol() string {
	switch op {
	case Set:
		return "="
	case Increment:
		return "++"
	case Decrement:
		return "--"
	case IncrementBy:
		return "+="
	case DecrementBy:
		return "-="
	default:
		panic(fmt.Sprintf("unknown assignment operation: %d", op))
	}
}

func (op AssignmentOperation) String() string {
	switch op {
	case Set:
		return "SET"
	case Increment:
		return "INCREMENT_ONE"
	case Decrement:
		return "DECREMENT_ONE"
	case IncrementBy:
		return "INCREMENT_AMOUNT"
	case DecrementBy:
		return "DECREMENT_AMOUNT"
	default:
		panic(fmt.Sprintf("unknown assignment operation: %d", op))
	}
}
