////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

type Params interface {
	GetDB() DB
	GetGroups() Groups
	GetPath() Path
	GetContext() Context
	GetReg() Reg
}

type paramsImpl struct {
	db      DB
	groups  Groups
	path    Path
	context Context
	reg     Reg
}

func NewParams(db DB, groups Groups, path Path, context Context, reg Reg) Params {
	return paramsImpl{
		db:      db,
		groups:  groups,
		path:    path,
		context: context,
		reg:     reg,
	}
}

func (params paramsImpl) GetDB() DB {
	return params.db
}

func (params paramsImpl) GetGroups() Groups {
	return params.groups
}

func (params paramsImpl) GetPath() Path {
	return params.path
}

func (params paramsImpl) GetContext() Context {
	return params.context
}

func (params paramsImpl) GetReg() Reg {
	return params.reg
}
