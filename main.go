/*
   @brief:
   @date:2022/7/17
*/
package read_go_zero

import "container/list"

const (

)

type ServiceGroup struct {
	ServiceList []Service	//服务列表
}

type Service interface {
	Name() string		//服务名称
	IsOpen() bool		//服务是否开启
	Start() (Service,error)		//服务开启
	Stop() error		//服务停止
}

func NewServiceGroup() *ServiceGroup {
	return &ServiceGroup{ServiceList:}
}


func (s *ServiceGroup)Start(f func(service Service, err error)) {

}

func (s *ServiceGroup)Stop() {
	for i := len(s.ServiceList) - 1; i >= 0; i-- {
		v.Stop()
	}
}