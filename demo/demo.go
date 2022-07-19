package main

import (
	"context"
	"google.golang.org/appengine/log"
	"os"
	"os/signal"
	"syscall"
)

var serverList = []Server{
	0: &Server1{},
	1: &Server2{},
}

type Server interface {
	Start(ctx context.Context) (Server, error)	//服务启动
	ID() int								//返回服务标志，用于循序启动
	Close() error	//关闭服务
	IsOpen() bool	//是否开启
}

//开启服务
func Start(ctx context.Context) {
	for _, v := range serverList {
		v.Start(ctx)
	}
}

type ServerNum uint8

const (
	DemoA ServerNum = iota
	DemoB
	MaxServerNum
)

//服务管理
type Manager struct {

}

func Loop() {
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)
	select {
		case sig := <- exit:
			log.Infof(context.Background(), "recv signal %s", sig.String())
	}
}

//服务启动入口
func (m *Manager)Start(ctx context.Context, s Server) error {
	if s == nil || !s.IsOpen() {	//未设置对象或未开启则直接退出
		return nil
	}
	v, err := s.Start(ctx)
	if err != nil {
		panic(err)
	}

	key := v.ID()	//构建k/v进行服务句柄存储
	//TODO 这里使用 封装的context
	return nil
}

type Server1 struct {

}

func (s *Server1) Start(ctx context.Context) (Server, error) {
	//Server1 启动
	return s, nil
}

func (s *Server1) ID() int {
	return 0
}

func (s *Server1) Close() error {
	//服务关闭
	return nil
}

func (s *Server1) IsOpen() bool {
	return true
}

var _ Server = (*Server1)(nil)

type Server2 struct {

}

func (s *Server2) Start(ctx context.Context) (Server, error) {
	//Server2 服务开启
	return s, nil
}

func (s *Server2) ID() int {
	return 1
}

func (s *Server2) Close() error {
	//服务关闭
	return nil
}

func (s *Server2) IsOpen() bool {
	return false
}

var _ Server = (*Server2)(nil)
