package testdata

// SimpleStruct 用于测试的简单结构体
type SimpleStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// SimpleInterface 用于测试的简单接口
type SimpleInterface interface {
	DoSomething(name string) error
	GetValue() int
}

// SimpleFunction 用于测试的简单函数
func SimpleFunction(a int, b string) (int, error) {
	if a > 0 {
		return a, nil
	}
	return 0, nil
}
