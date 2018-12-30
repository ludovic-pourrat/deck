package state

import (
	memdb "github.com/hashicorp/go-memdb"
	"github.com/pkg/errors"
)

const (
	routeTableName = "route"
)

var routeTableSchema = &memdb.TableSchema{
	Name: routeTableName,
	Indexes: map[string]*memdb.IndexSchema{
		id: {
			Name:    id,
			Unique:  true,
			Indexer: &memdb.StringFieldIndex{Field: "ID"},
		},
		// TODO add ServiceName/ServiceID both fields for indexing
		"routesByServiceName": {
			Name: "routesByServiceName",
			Indexer: &SubFieldIndexer{
				StructField: "Service",
				SubField:    "Name",
			},
		},
		"routesByServiceID": {
			Name: "routesByServiceID",
			Indexer: &SubFieldIndexer{
				StructField: "Service",
				SubField:    "ID",
			},
		},
		"name": {
			Name:    "name",
			Unique:  true,
			Indexer: &memdb.StringFieldIndex{Field: "Name"},
		},
		all: {
			Name: all,
			Indexer: &memdb.ConditionalIndex{
				Conditional: func(v interface{}) (bool, error) {
					return true, nil
				},
			},
		},
	},
}

// RoutesCollection stores and indexes Kong Services.
type RoutesCollection struct {
	memdb *memdb.MemDB
}

// NewRoutesCollection instantiates a RoutesCollection.
func NewRoutesCollection() (*RoutesCollection, error) {
	var schema = &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			routeTableName: routeTableSchema,
		},
	}
	m, err := memdb.NewMemDB(schema)
	if err != nil {
		return nil, errors.Wrap(err, "creating new RouteCollection")
	}
	return &RoutesCollection{
		memdb: m,
	}, nil
}

// Add adds a route to RoutesCollection
func (k *RoutesCollection) Add(route Route) error {
	txn := k.memdb.Txn(true)
	defer txn.Abort()
	err := txn.Insert(routeTableName, &route)
	if err != nil {
		return errors.Wrap(err, "insert failed")
	}
	txn.Commit()
	return nil
}

// Get gets a route by name or ID.
func (k *RoutesCollection) Get(ID string) (*Route, error) {
	res, err := multiIndexLookup(k.memdb, routeTableName, []string{"name", id}, ID)
	if err == ErrNotFound {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, errors.Wrap(err, "route lookup failed")
	}
	if res == nil {
		return nil, ErrNotFound
	}
	route, ok := res.(*Route)
	if !ok {
		panic("unexpected type found")
	}
	return route, nil
}

// GetAllRoutesByServiceName returns all routes referencing a service
// by its name.
func (k *RoutesCollection) GetAllRoutesByServiceName(name string) ([]*Route, error) {
	txn := k.memdb.Txn(false)
	iter, err := txn.Get(routeTableName, "routesByServiceName", name)
	if err != nil {
		return nil, err
	}
	var res []*Route
	for el := iter.Next(); el != nil; el = iter.Next() {
		s, ok := el.(*Route)
		if !ok {
			panic("unexpected type found")
		}
		res = append(res, s)
	}
	return res, nil
}

// GetAllRoutesByServiceID returns all routes referencing a service
// by its id.
func (k *RoutesCollection) GetAllRoutesByServiceID(id string) ([]*Route, error) {
	txn := k.memdb.Txn(false)
	iter, err := txn.Get(routeTableName, "routesByServiceID", id)
	if err != nil {
		return nil, err
	}
	var res []*Route
	for el := iter.Next(); el != nil; el = iter.Next() {
		s, ok := el.(*Route)
		if !ok {
			panic("unexpected type found")
		}
		res = append(res, s)
	}
	return res, nil
}

// Update updates a route
func (k *RoutesCollection) Update(route Route) error {
	txn := k.memdb.Txn(true)
	defer txn.Abort()
	err := txn.Insert(routeTableName, &route)
	if err != nil {
		return errors.Wrap(err, "update failed")
	}
	txn.Commit()
	return nil
}

// Delete deletes a route by name or ID.
func (k *RoutesCollection) Delete(nameOrID string) error {
	route, err := k.Get(nameOrID)

	if err != nil {
		return errors.Wrap(err, "looking up route")
	}

	txn := k.memdb.Txn(true)
	defer txn.Abort()

	err = txn.Delete(routeTableName, route)
	if err != nil {
		return errors.Wrap(err, "delete failed")
	}
	txn.Commit()
	return nil
}

// GetAll gets a route by name or ID.
func (k *RoutesCollection) GetAll() ([]*Route, error) {
	txn := k.memdb.Txn(false)
	defer txn.Abort()

	iter, err := txn.Get(routeTableName, all, true)
	if err != nil {
		return nil, errors.Wrapf(err, "route lookup failed")
	}

	var res []*Route
	for el := iter.Next(); el != nil; el = iter.Next() {
		s, ok := el.(*Route)
		if !ok {
			panic("unexpected type found")
		}
		res = append(res, s)
	}
	txn.Commit()
	return res, nil
}