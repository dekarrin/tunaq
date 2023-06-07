package syntax

import "fmt"

type BinaryOperation int

const (
	OpBinaryEqual BinaryOperation = iota
	OpBinaryNotEqual
	OpBinaryLessThan
	OpBinaryLessThanEqual
	OpBinaryGreaterThan
	OpBinaryGreaterThanEqual
	OpBinaryAdd
	OpBinarySubtract
	OpBinaryDivide
	OpBinaryMultiply
	OpBinaryLogicalAnd
	OpBinaryLogicalOr
)

// Symbol returns the Tunascript operator that corresponds to the operation.
func (op BinaryOperation) Symbol() string {
	switch op {
	case OpBinaryEqual:
		return "="
	case OpBinaryNotEqual:
		return "!="
	case OpBinaryLessThan:
		return "<"
	case OpBinaryLessThanEqual:
		return "<="
	case OpBinaryGreaterThan:
		return ">"
	case OpBinaryGreaterThanEqual:
		return ">="
	case OpBinaryAdd:
		return "+"
	case OpBinarySubtract:
		return "-"
	case OpBinaryDivide:
		return "/"
	case OpBinaryMultiply:
		return "*"
	case OpBinaryLogicalAnd:
		return "&&"
	case OpBinaryLogicalOr:
		return "||"
	default:
		panic(fmt.Sprintf("unknown binary operation: %d", op))
	}
}

func (op BinaryOperation) String() string {
	switch op {
	case OpBinaryEqual:
		return "EQUALITY"
	case OpBinaryNotEqual:
		return "NON_EQUALITY"
	case OpBinaryLessThan:
		return "LESS_THAN"
	case OpBinaryLessThanEqual:
		return "LESS_THAN_EQUALITY"
	case OpBinaryGreaterThan:
		return "GREATER_THAN"
	case OpBinaryGreaterThanEqual:
		return "GREATER_THAN_EQUALITY"
	case OpBinaryAdd:
		return "ADDITION"
	case OpBinarySubtract:
		return "SUBTRACTION"
	case OpBinaryDivide:
		return "DIVISION"
	case OpBinaryMultiply:
		return "MULTIPLICATION"
	case OpBinaryLogicalAnd:
		return "AND"
	case OpBinaryLogicalOr:
		return "OR"
	default:
		panic(fmt.Sprintf("unknown binary operation: %d", op))
	}
}

type UnaryOperation int

const (
	OpUnaryNegate UnaryOperation = iota
	OpUnaryLogicalNot
)

// Symbol returns the Tunascript operator that corresponds to the operation.
func (op UnaryOperation) Symbol() string {
	switch op {
	case OpUnaryNegate:
		return "-"
	case OpUnaryLogicalNot:
		return "!"
	default:
		panic(fmt.Sprintf("unknown unary operation: %d", op))
	}
}

// String returns the Tunascript operator that corresponds to the operation.
func (op UnaryOperation) String() string {
	switch op {
	case OpUnaryNegate:
		return "NEGATION"
	case OpUnaryLogicalNot:
		return "NOT"
	default:
		panic(fmt.Sprintf("unknown unary operation: %d", op))
	}
}

type AssignmentOperation int

const (
	OpAssignSet AssignmentOperation = iota
	OpAssignIncrement
	OpAssignDecrement
	OpAssignIncrementBy
	OpAssignDecrementBy
)

// Symbol returns the Tunascript operator that corresponds to the operation.
func (op AssignmentOperation) Symbol() string {
	switch op {
	case OpAssignSet:
		return "="
	case OpAssignIncrement:
		return "++"
	case OpAssignDecrement:
		return "--"
	case OpAssignIncrementBy:
		return "+="
	case OpAssignDecrementBy:
		return "-="
	default:
		panic(fmt.Sprintf("unknown assignment operation: %d", op))
	}
}

func (op AssignmentOperation) String() string {
	switch op {
	case OpAssignSet:
		return "SET"
	case OpAssignIncrement:
		return "INCREMENT_ONE"
	case OpAssignDecrement:
		return "DECREMENT_ONE"
	case OpAssignIncrementBy:
		return "INCREMENT_AMOUNT"
	case OpAssignDecrementBy:
		return "DECREMENT_AMOUNT"
	default:
		panic(fmt.Sprintf("unknown assignment operation: %d", op))
	}
}
