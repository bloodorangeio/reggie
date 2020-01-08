package reggie

type (
	ErrorBody struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Detail  string `json:"detail"`
	}

	//TODO: import from opencontainers/distribution-spec/specs-go/v1
	Error struct {
		Details []ErrorBody `json:"errors"`
	}
)

func (e *Error) Code() string {
	return e.Details[0].Code
}

func (e *Error) Message() string {
	return e.Details[0].Message
}

func (e *Error) Detail() string {
	return e.Details[0].Detail
}
