package resource

import (
	"github.com/kyverno/kyverno/cmd/cli/kubectl-kyverno/resource/loader"
	"github.com/kyverno/kyverno/ext/resource/convert"
)

func Load[T any](l loader.Loader, content []byte) (*T, error) {
	_, untyped, err := l.Load(content)
	if err != nil {
		return nil, err
	}
	result, err := convert.To[T](untyped)
	if err != nil {
		return nil, err
	}
	return result, nil
}
