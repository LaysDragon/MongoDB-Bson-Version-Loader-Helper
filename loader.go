package loader

import (
	"github.com/ompluscator/dynamic-struct"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"
	"reflect"
)

//LoaderNotFoundError while can't found loader for the specified version
var LoaderNotFoundError = xerrors.New("cannot found loader handlers for src version")

//TransformerNotFoundError while can't found Transformer for specified version path transformation progress
var TransformerNotFoundError = xerrors.New("cannot found next transformer to the target version")

//TransformerSrcTypeIncorrectError while Transformer trying to cast interface{} src data into desired version struct and failed
var TransformerSrcTypeIncorrectError = xerrors.New("src type for transformer is incorrect and failed to cast")

//NoVersionTagError while loaded data not contain any valid _version tag
var NoVersionTagError = xerrors.New("data has no _version field or with invalid 0.0 version")

//VersionCapture a Capture for extract the version info from byte data and carry the corresponding version structure data though the progress
type VersionCapture struct {
	Version Version `bson:"_version"`
	Data    interface{}
}

//AVersionCapture use for prevent MarshalBSON loop
type AVersionCapture VersionCapture

//MarshalBSON where VersionCapture deal with version and inline Data
func (v VersionCapture) MarshalBSON() ([]byte, error) {
	if v.GetData() != nil {
		versionWrapper := GetVersionWrapperStruct(v.GetData()).New()
		reflect.ValueOf(versionWrapper).Elem().FieldByName("Data").Set(reflect.ValueOf(v.GetData()))
		reflect.ValueOf(versionWrapper).Elem().FieldByName("Version").Set(reflect.ValueOf(v.GetVersion()))

		return bson.Marshal(versionWrapper)
	}
	return bson.Marshal(AVersionCapture(v))
}

//UnmarshalBSON  where VersionCapture deal with version and inline Data
func (v *VersionCapture) UnmarshalBSON(src []byte) error {
	if v.GetData() != nil {
		versionWrapper := GetVersionWrapperStruct(v.GetData()).New()
		if err := bson.Unmarshal(src, versionWrapper); err != nil {
			return err
		}

		reader := dynamicstruct.NewReader(versionWrapper)
		v.SetVersion(reader.GetField("Version").Interface().(Version))
		v.SetData(reader.GetField("Data").Interface())
	}
	return bson.Unmarshal(src, (*AVersionCapture)(v))
}

//SetVersion set Version
func (v *VersionCapture) SetVersion(vv Version) {
	v.Version = vv
}

//GetVersion get Version
func (v VersionCapture) GetVersion() Version {
	return v.Version
}

//SetData set Data
func (v *VersionCapture) SetData(d interface{}) {
	v.Data = d
}

//GetData get Data
func (v VersionCapture) GetData() interface{} {
	return v.Data
}

//Transformer func that responsible for transform the data between Versions
type Transformer func(VersionWrapper) error

type TargetTransformers map[Version]Transformer
type SrcToTargetTransformers map[Version]TargetTransformers

type SrcToTargetVersions map[Version]Versions

//Loader func that responsible for load data for specified Version
type Loader func([]byte, VersionWrapper) error

type SrcLoaders map[Version]Loader

type Registry struct {
	loaders      SrcLoaders
	transformers SrcToTargetTransformers
	versions     SrcToTargetVersions
}

func (l *Registry) add(src Version, target Version, loader Transformer) {
	targetLoaders, ok := l.transformers[src]
	if !ok {
		targetLoaders = TargetTransformers{}
		l.transformers[src] = targetLoaders
	}
	targetVersions, ok := l.versions[src]
	if !ok {
		targetVersions = []Version{}
		l.versions[src] = targetVersions
	}
	targetLoaders[target] = loader
	l.versions[src] = append(targetVersions, target)
}

//type SLoaders map[string]Loader
type SLoaders map[string]interface{}
type STransformers map[string]STargetTransformers
type STargetTransformers map[string]Transformer

func (l SLoaders) SrcLoaders() SrcLoaders {
	s := SrcLoaders{}
	for v, l := range l {
		if loader, ok := l.(Loader); ok {
			s[NewVersionPanic(v)] = loader
		} else {
			s[NewVersionPanic(v)] = DefaultLoader(l)
		}

	}
	return s
}

func (t STransformers) SrcToTargetTransformers() SrcToTargetTransformers {
	s := SrcToTargetTransformers{}
	for src, t := range t {
		s[NewVersionPanic(src)] = TargetTransformers{}
		for tv, tt := range t {
			s[NewVersionPanic(src)][NewVersionPanic(tv)] = tt
		}
	}
	return s
}

var DynamicVersionWrapperStructs = map[reflect.Type]dynamicstruct.DynamicStruct{}

func AddVersionWrapperType(typeVal interface{}) dynamicstruct.DynamicStruct {
	wrapper := dynamicstruct.NewStruct().
		AddField("Version", Version{}, `bson:"_version"`).
		AddField("Data", typeVal, `bson:",inline"`).
		Build()
	DynamicVersionWrapperStructs[reflect.TypeOf(typeVal)] = wrapper
	return wrapper
}

func GetVersionWrapperStruct(typeVal interface{}) dynamicstruct.DynamicStruct {
	if wrapper, ok := DynamicVersionWrapperStructs[reflect.TypeOf(typeVal)]; ok {
		return wrapper
	} else {
		return AddVersionWrapperType(typeVal)
	}

}

//NewRegistry Registry the Loaders and Transformers for migration
func NewRegistry(loadersL SLoaders, transformersT STransformers) *Registry {
	l := &Registry{
		SrcLoaders{},
		SrcToTargetTransformers{},
		SrcToTargetVersions{},
	}
	loaders := loadersL.SrcLoaders()
	transformers := transformersT.SrcToTargetTransformers()
	for version, loader := range loaders {
		l.loaders[version] = loader
	}
	for srcVersion, targetTransformers := range transformers {
		l.transformers[srcVersion] = targetTransformers
		l.versions[srcVersion] = Versions{}
		for targetVersion := range targetTransformers {
			l.versions[srcVersion] = append(l.versions[srcVersion], targetVersion)
		}
	}

	return l
}

//Transform transformation the src Data Struct to target Version
func (l *Registry) Transform(data VersionWrapper, target Version) error {

	if data.GetVersion().Greater(target) {
		return xerrors.Errorf("Raise error from trying donwngrading version %s to %s for %STransformers,please update your target struct version to lastest:%w", data.GetVersion(), target, data, TransformerNotFoundError)
	}
	for data.GetVersion() != target {

		targetVersions, ok := l.versions[data.GetVersion()]
		if !ok {
			return xerrors.Errorf("Raise error from version %s to %s for %STransformers:%w", data.GetVersion(), target, data, TransformerNotFoundError)
		}
		targetTransformers, ok := l.transformers[data.GetVersion()]
		if !ok {
			return xerrors.Errorf("Raise error from version %s to %s for %STransformers:%w", data.GetVersion(), target, data, TransformerNotFoundError)
		}

		var nextVersion Version
		if _, ok := targetTransformers[target]; ok {
			nextVersion = target
		} else {
			nextVersion = *targetVersions.Max()
		}
		if err := targetTransformers[nextVersion](data); err != nil {
			return err
		}
	}
	return nil

}

//Load load the bson src bytes into target Version,it will return a Version Container with data
func (l *Registry) Load(src []byte, target Version) (VersionWrapper, error) {
	versionCapture := VersionCapture{}
	if err := bson.Unmarshal(src, &versionCapture); err != nil {
		return nil, err
	}
	if (VersionCapture{}) == versionCapture {
		return nil, xerrors.Errorf("Raise error %w", NoVersionTagError)
	}
	var processingTarget VersionWrapper
	for version, loader := range l.loaders {
		if version == versionCapture.Version {
			processingTarget = &VersionCapture{}
			if err := loader(src, processingTarget); err != nil {
				return nil, xerrors.Errorf("Raise error while trying to load data:%w", err)
			}
			break
		}
	}
	if processingTarget == nil {
		return nil, xerrors.Errorf("Raise error from src version %s:%w", versionCapture.Version, LoaderNotFoundError)
	}
	if err := l.Transform(processingTarget, target); err != nil {
		return nil, err
	}
	return processingTarget, nil
}

func DefaultLoader(typeVal interface{}) Loader {
	typ := reflect.TypeOf(typeVal)
	return func(src []byte, dst VersionWrapper) error {
		dst.SetData(reflect.New(typ).Elem().Interface())

		if err := bson.Unmarshal(src, dst); err != nil {
			return err
		}
		return nil
	}
}
