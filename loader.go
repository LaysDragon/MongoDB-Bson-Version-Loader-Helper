package loader

import (
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"
)

var LoaderNotFound = xerrors.New("cannot found loader handlers for src version")
var TransformerNotFound = xerrors.New("cannot found next transformer to the target version")
var LoaderSrcTypeIncorrect = xerrors.New("cannot cast HasVersion to target version type")
var NoVersionTag = xerrors.New("data not container version field")

type D interface{}
type VersionCapture struct {
	Version Version
	D
}

func (v *VersionCapture) SetVersion(vv Version) {
	v.Version = vv
}

func (v VersionCapture) GetData() interface{} {
	return v.D
}

func (v *VersionCapture) SetData(d interface{}) {
	v.D = d
}

func (v VersionCapture) GetVersion() Version {
	return v.Version
}

type Transformer func(HasVersion) error

type TargetTransformers map[Version]Transformer
type SrcToTargetTransformers map[Version]TargetTransformers

type SrcToTargetVersions map[Version]Versions

type Loader func([]byte, HasVersion) error

type SrcLoaders map[Version]Loader

type LoaderRegistry struct {
	loaders      SrcLoaders
	transformers SrcToTargetTransformers
	versions     SrcToTargetVersions
}

func (l *LoaderRegistry) add(src Version, target Version, loader Transformer) {
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

type SLoaders map[string]Loader
type STransformers map[string]STargetTransformers
type STargetTransformers map[string]Transformer

func (l SLoaders) SrcLoaders() SrcLoaders {
	s := SrcLoaders{}
	for v, l := range l {
		s[NewVersionPanic(v)] = l
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

func NewLoaderRegistry(loadersL SLoaders, transformersT STransformers) *LoaderRegistry {
	l := &LoaderRegistry{
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

func (l *LoaderRegistry) Transform(data HasVersion, target Version) error {

	if data.GetVersion().Greater(target) {
		return xerrors.Errorf("Raise error from trying donwngrading version %s to %s for %STransformers,please update your target struct version to lastest:%w", data.GetVersion(), target, data, TransformerNotFound)
	}
	for data.GetVersion() != target {

		targetVersions, ok := l.versions[data.GetVersion()]
		if !ok {
			return xerrors.Errorf("Raise error from version %s to %s for %STransformers:%w", data.GetVersion(), target, data, TransformerNotFound)
		}
		targetTransformers, ok := l.transformers[data.GetVersion()]
		if !ok {
			return xerrors.Errorf("Raise error from version %s to %s for %STransformers:%w", data.GetVersion(), target, data, TransformerNotFound)
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

func (l *LoaderRegistry) Load(src []byte, target Version) (HasVersion, error) {
	versionCapture := VersionCapture{
		D: struct{}{},
	}
	if err := bson.Unmarshal(src, &versionCapture); err != nil {
		return nil, err
	}
	if (VersionCapture{}) == versionCapture {
		return nil, xerrors.Errorf("Raise error %w", NoVersionTag)
	}
	var processingTarget HasVersion
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
		return nil, xerrors.Errorf("Raise error from src version %s:%w", versionCapture.Version, LoaderNotFound)
	}
	if err := l.Transform(processingTarget, target); err != nil {
		return nil, err
	}
	return processingTarget, nil
}