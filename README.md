# chainnode


## to add support to chain X
1. Add package X inside package `chainadaptor` and implement the interface [`chainadaptor.ChainAdaptor`](chainadaptor/chainadaptor.go) (embed the
 [`fallback.ChainAdaptor`](chainadaptor/fallback/adaptor.go) and override specific methods).

2. Provide `NewXChainAdaptor` factory method and register it in the dispatcher
