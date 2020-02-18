package main

type testConfigService struct {
	handler *APIHandler
	addr    string
}

func (t *testConfigService) launch(handler *APIHandler, addr string) {
	t.handler = handler
	t.addr = addr
}

func (t *testConfigService) stop() {
	// nothing
}
