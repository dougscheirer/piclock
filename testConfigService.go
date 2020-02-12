package main

type testConfigService struct {
	handler *apiHandler
	addr    string
}

func (t *testConfigService) launch(handler *apiHandler, addr string) {
	t.handler = handler
	t.addr = addr
}

func (t *testConfigService) stop() {
	// nothing
}
