package security

type filter interface {
	check(string, string) error
}

type Filter struct {
	filters []filter
}

func NewFilter() *Filter {
	return &Filter{}
}

// 注册过滤器
func (f *Filter) Register(filter filter) {
	f.filters = append(f.filters, filter)
}

//
func (f *Filter) Check(ip string, pubKey string) error {
	for _, filter := range f.filters {
		if err := filter.check(ip, pubKey)；; err != nil {
			return nil
		}
	}

	return nil
}
