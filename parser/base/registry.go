package base

import (
	"fmt"
)

type FunctionRegistry struct {
	serverFn        map[string]ServerFunction
	serverFnAlias   map[string]string
	userFn          map[string]UserFunction
	userFnAlias     map[string]string
	databaseFn      map[string]DatabaseFunction
	databaseFnAlias map[string]string
}

func (r *FunctionRegistry) NewServerItem(name, alias string, fn ServerFunction) {
	// cehck if entry already exists
	if _, ok := r.serverFn[name]; ok {
		panic(fmt.Errorf("Server function %s already registered", name))
	}
	// register function
	r.serverFn[name] = fn

	// register alias
	if alias != "" {
		if alias != name {
			if _, ok := r.serverFnAlias[alias]; ok {
				panic(fmt.Errorf("Server function %s alias already registered", name))
			}
			r.serverFnAlias[alias] = name
		} else {
			panic(fmt.Errorf("Server function %s alias is invalid", name))
		}
	}
}

func (r *FunctionRegistry) GetServerItem(name string) (ServerFunction, string) {
	fn, ok := r.serverFn[name]
	if !ok {
		// check alias
		name, ok = r.serverFnAlias[name]
		if !ok {
			return nil, ""
		}
		return r.GetServerItem(name)
	}
	return fn, name
}

func (r *FunctionRegistry) NewUserItem(name, alias string, fn UserFunction) {
	// cehck if entry already exists
	if _, ok := r.userFn[name]; ok {
		panic(fmt.Errorf("User function %s already registered", name))
	}
	// register function
	r.userFn[name] = fn

	// register alias
	if alias != "" {
		if alias != name {
			if _, ok := r.userFnAlias[alias]; ok {
				panic(fmt.Errorf("User function %s alias already registered", name))
			}
			r.userFnAlias[alias] = name
		} else {
			panic(fmt.Errorf("User function %s alias is invalid", name))
		}
	}
}

func (r *FunctionRegistry) GetUserItem(name string) (UserFunction, string) {
	fn, ok := r.userFn[name]
	if !ok {
		// check alias
		name, ok = r.userFnAlias[name]
		if !ok {
			return nil, ""
		}
		return r.GetUserItem(name)
	}
	return fn, name
}

func (r *FunctionRegistry) NewDatabaseItem(name, alias string, fn DatabaseFunction) {
	// cehck if entry already exists
	if _, ok := r.databaseFn[name]; ok {
		panic(fmt.Errorf("Database function %s already registered", name))
	}
	// register function
	r.databaseFn[name] = fn

	// register alias
	if alias != "" {
		if alias != name {
			if _, ok := r.databaseFnAlias[alias]; ok {
				panic(fmt.Errorf("Database function %s alias already registered", name))
			}
			r.databaseFnAlias[alias] = name
		} else {
			panic(fmt.Errorf("Database function %s alias is invalid", name))
		}
	}
}

func (r *FunctionRegistry) GetDatabaseItem(name string) (DatabaseFunction, string) {
	fn, ok := r.databaseFn[name]
	if !ok {
		// check alias
		name, ok = r.databaseFnAlias[name]
		if !ok {
			return nil, ""
		}
		return r.GetDatabaseItem(name)
	}
	return fn, name
}

func NewFunctionRegistry() *FunctionRegistry {
	fr := FunctionRegistry{
		serverFn:        make(map[string]ServerFunction),
		serverFnAlias:   make(map[string]string),
		userFn:          make(map[string]UserFunction),
		userFnAlias:     make(map[string]string),
		databaseFn:      make(map[string]DatabaseFunction),
		databaseFnAlias: make(map[string]string),
	}
	return &fr
}
