package main

type testConfigService struct {
	handler *myHandler
	addr    string
}

func (t *testConfigService) launch(handler *myHandler, addr string) {
	t.handler = handler
	t.addr = addr
}

func (t *testConfigService) stop() {
	// nothing
}
