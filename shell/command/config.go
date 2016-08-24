package command

import (
	"github.com/qmsk/close/shell/config"
	"reflect"
)

type GenericConfig interface {
	Path()      string
	ResType()   reflect.Type
	FieldName() string
}

type GenericConfigImpl struct {
	path      string

	resType   reflect.Type
	fieldName string
}

func (config GenericConfigImpl) Command(options config.CommonOptions) (config.Command, error) {
	genericCommand := &GenericCommandImpl{
		url: options.Url(),
		user: options.User(),
		config: config,
	}

	return genericCommand, nil
}

func NewGenericConfigImpl(path string, resType reflect.Type, fieldName string) *GenericConfigImpl {
	config := &GenericConfigImpl {}
	config.init(path, resType, fieldName)
	return config
}

func (config *GenericConfigImpl) init(path string, resType reflect.Type, fieldName string) {
	config.path = path
	config.resType = resType
	config.fieldName = fieldName
}

func (config GenericConfigImpl) Path() string { return config.path }
func (config GenericConfigImpl) ResType() reflect.Type { return config.resType }
func (config GenericConfigImpl) FieldName() string { return config.fieldName }
