package newrelic

import (
	"net/http"
	"path/filepath"
	"runtime"
	"sync"

	"golang.org/x/net/context"
)

// Tx represents a transaction.
type Tx interface {
	Start() error
	End() error
	StartGeneric(name string) error
	StartDatastore(table, operation, sql, rollupName string) error
	StartExternal(host, name string) error
	EndSegment() error
	ReportError(exceptionType, errorMessage, stackTrace, stackFrameDelim string) error
}

// tx implements the Tx interface.
type tx struct {
	Tracer   TxTracer
	Reporter TxReporter

	id       int64
	name     string
	url      string
	category string
	txnType  TransactionType
	ss       *SegmentStack

	mtx *sync.Mutex
}

// NewTx returns a new transaction.
func NewTx(name string) *tx {
	return &tx{
		Tracer:   &NRTxTracer{},
		Reporter: &NRTxReporter{},
		name:     name,
		txnType:  WebTransaction,
		ss:       NewSegmentStack(),
		mtx:      &sync.Mutex{},
	}
}

// NewRequestTx returns a new transaction with a request url.
func NewRequestTx(name string, url string) *tx {
	t := NewTx(name)
	t.url = url
	return t
}

// NewBackgroundTx returns a new background transaction
func NewBackgroundTx(name string, category string) *tx {
	t := NewTx(name)
	t.txnType = OtherTransaction
	t.category = category
	return t
}

// Start starts a transaction, setting the id.
func (t *tx) Start() (err error) {
	if t.id != 0 {
		return ErrTxAlreadyStarted
	}
	if t.id, err = t.Tracer.BeginTransaction(); err != nil {
		return err
	}
	if err = t.Tracer.SetTransactionName(t.id, t.name); err != nil {
		return err
	}
	if err = t.Tracer.SetTransactionType(t.id, t.txnType); err != nil {
		return err
	}
	if t.url != "" {
		if err = t.Tracer.SetTransactionRequestURL(t.id, t.url); err != nil {

			return err
		}
	}
	if t.category != "" {
		if err = t.Tracer.SetTransactionCategory(t.id, t.category); err != nil {
			return err
		}
	}

	return nil
}

// End ends a transaction.
func (t *tx) End() error {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	for t.ss.Peek() != rootSegment {
		t.EndSegment() // discarding errors?
	}
	return t.Tracer.EndTransaction(t.id)
}

// StartGeneric starts a generic segment.
func (t *tx) StartGeneric(name string) error {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	id, err := t.Tracer.BeginGenericSegment(t.id, t.ss.Peek(), name)
	if err != nil {
		return err
	}
	t.ss.Push(id)
	return nil
}

// StartDatastore starts a datastore segment.
func (t *tx) StartDatastore(table, operation, sql, rollupName string) error {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	id, err := t.Tracer.BeginDatastoreSegment(t.id, t.ss.Peek(), table, operation, sql, rollupName)
	if err != nil {
		return err
	}
	t.ss.Push(id)
	return nil
}

// StartExternal starts an external segment.
func (t *tx) StartExternal(host, name string) error {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	id, err := t.Tracer.BeginExternalSegment(t.id, t.ss.Peek(), host, name)
	if err != nil {
		return err
	}
	t.ss.Push(id)
	return nil
}

// EndSegment ends the segment at the top of the stack.
func (t *tx) EndSegment() error {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if id, ok := t.ss.Pop(); ok {
		return t.Tracer.EndSegment(t.id, id)
	}
	return nil
}

// ReportError reports an error that occured during the transaction.
func (t *tx) ReportError(exceptionType, errorMessage, stackTrace, stackFrameDelim string) error {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	_, err := t.Reporter.ReportError(t.id, exceptionType, errorMessage, stackTrace, stackFrameDelim)
	return err
}

// WithTx inserts a newrelic.Tx into the provided context.
func WithTx(ctx context.Context, t Tx) context.Context {
	return context.WithValue(ctx, txKey, t)
}

// FromContext returns a newrelic.Tx from the context.
func FromContext(ctx context.Context) (Tx, bool) {
	t, ok := ctx.Value(txKey).(Tx)
	return t, ok
}

type Trace struct {
	err  error
	done func() error
}

func (t *Trace) Err() error {
	return t.err
}

func (t *Trace) Done() {
	if t.err == nil {
		t.err = t.done()
	}
}

// TraceReqest traces an http request. It returns a new context with the transaction
// included in it, and a trace object.
//
// Usage:
//
//     ctx, t := TraceRequest(ctx, name, req)
//     defer t.Done()
func TraceRequest(ctx context.Context, name string, req *http.Request) (context.Context, *Trace) {
	tx := NewRequestTx(name, req.URL.String())
	ctx = WithTx(ctx, tx)
	err := tx.Start()

	return ctx, &Trace{
		err: err,
		done: func() error {
			return tx.End()
		},
	}
}

// TraceExternal adds an external segment to the newrelic transaction, if one exists in the context.
func TraceExternal(ctx context.Context, host, name string) *Trace {
	return trace(ctx, name, func(tx Tx) error {
		return tx.StartExternal(host, name)
	})
}

// TraceGeneric adds a generic segment to the newrelic transaction, if one exists in the context.
func TraceGeneric(ctx context.Context, name string) *Trace {
	return trace(ctx, name, func(tx Tx) error {
		return tx.StartGeneric(name)
	})
}

// TraceDatastore adds a datastore segment to the newrelic transaction, if one exists in the context.
func TraceDatastore(ctx context.Context, table, operation, sql, rollupName string) *Trace {
	return trace(ctx, rollupName, func(tx Tx) error {
		return tx.StartDatastore(table, operation, sql, rollupName)
	})
}

// TraceFunc adds a generic segment, autodetecting the function name with runtime.Caller().
func TraceFunc(ctx context.Context) *Trace {
	name := caller(2) // Get the caller that called TraceFunc.
	return trace(ctx, name, func(tx Tx) error {
		return tx.StartGeneric(name)
	})
}

// trace is a helper function for TraceExternal and TraceGeneric, you probably don't want
// to use it directly.
func trace(ctx context.Context, name string, fn func(Tx) error) *Trace {
	if tx, ok := FromContext(ctx); ok {
		err := fn(tx)
		return &Trace{
			err: err,
			done: func() error {
				return tx.EndSegment()
			},
		}
	}
	return &Trace{nil, func() error { return nil }}
}

// caller returns the name of the function that called the function this function was called from.
// n = 1 => caller of caller()
// n = 2 => caller of caller of call()
// etc.
func caller(n int) string {
	name := "unknown"
	if pc, _, _, ok := runtime.Caller(n); ok {
		name = filepath.Base(runtime.FuncForPC(pc).Name())
	}
	return name
}

type key int

const (
	txKey key = iota
)
