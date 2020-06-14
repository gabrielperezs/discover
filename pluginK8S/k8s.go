package pluginK8S

import (
	"errors"
	"log"
	"time"

	"github.com/tevino/abool"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	ErrInvalidLogin = errors.New("Invalid login")
)

var (
	getPodsTimeout = int64(time.Duration(1 * time.Minute).Seconds())
)

type PluginK8S struct {
	C         chan []string
	t         *time.Timer
	cfg       Config
	namespace string
	watch     bool
	config    *rest.Config
	clientset *kubernetes.Clientset
	exit      *abool.AtomicBool
}

func New(c Config) *PluginK8S {
	l := &PluginK8S{
		C:    make(chan []string, 1),
		t:    time.NewTimer(c.Refresh),
		cfg:  c,
		exit: abool.New(),
	}
	if err := l.Reload(c); err != nil {
		log.Printf("ERROR: %+v", err)
	}
	go l.interval()
	return l
}

func (l *PluginK8S) Reload(c Config) (err error) {
	l.namespace = c.Namespace

	l.config, err = clientcmd.BuildConfigFromFlags(c.MasterURL, c.KubeConfigPath)
	if err != nil {
		return err
	}

	l.clientset, err = kubernetes.NewForConfig(l.config)
	if err != nil {
		return err
	}
	return
}

func (l *PluginK8S) Get() chan []string {
	return l.C
}

func (l *PluginK8S) Protocol() string {
	return l.cfg.Protocol
}

func (l *PluginK8S) Weight() int64 {
	return l.cfg.Weight
}

func (l *PluginK8S) Timeout() time.Duration {
	return l.cfg.Timeout
}

func (l *PluginK8S) Exit() {
	l.exit.Set()
	l.t.Stop()
	time.Sleep(l.cfg.Refresh + (time.Millisecond * 100))
	close(l.C)
}

func (l *PluginK8S) send(hosts []string) {
	if l.exit.IsSet() == false {
		l.C <- hosts
	}
}

func (l *PluginK8S) interval() {
	if hosts, err := l.once(); err == nil {
		l.send(hosts)
	}

	for range l.t.C {
		l.get()
		l.t.Reset(l.cfg.Refresh)
	}
}

func (l *PluginK8S) once() ([]string, error) {
	if l.clientset == nil {
		return nil, ErrInvalidLogin
	}

	pods, err := l.clientset.CoreV1().Pods(l.namespace).List(v1.ListOptions{
		Watch:          false,
		TimeoutSeconds: &getPodsTimeout,
	})
	if err != nil {
		return nil, err
	}

	listIP := make([]string, 0)
	for _, pod := range pods.Items {
		listIP = append(listIP, pod.Status.PodIP+":"+l.cfg.Port)
	}

	return listIP, nil
}

func (l *PluginK8S) get() {
	if l.clientset == nil {
		return
	}

	events := l.clientset.EventsV1beta1().Events(l.namespace)
	w, _ := events.Watch(v1.ListOptions{
		Watch:          true,
		TimeoutSeconds: &getPodsTimeout,
	})
	for {
		result := <-w.ResultChan()
		if result.Type == watch.Error {
			break
		}

		if hosts, err := l.once(); err == nil {
			l.send(hosts)
		}

		// cp, ok := result.Object.DeepCopyObject().(*v1beta1.Event)
		// if !ok {
		// 	break
		// }

		// //ncp := cp.(*watch.Event)
		// log.Printf("-----------------------------")
		// log.Printf("ObjectMeta: %#v", cp.ObjectMeta)
		// log.Printf("ReportingController: %#v", cp.ReportingController)
		// log.Printf("ReportingInstance: %#v", cp.ReportingInstance)
		// log.Printf("Action: %#v", cp.Action)
		// log.Printf("Reason: %#v", cp.Reason)
	}

	return
}
