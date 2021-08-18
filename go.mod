module github.com/ridecell/ridectl

go 1.16

require (
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.4.2 // indirect
	github.com/Masterminds/sprig v2.19.0+incompatible // indirect
	github.com/Ridecell/ridecell-operator v0.0.0-20200129211046-d955c632e46c
	github.com/Ridecell/ridectl v0.0.35
	github.com/apoorvam/goterminal v0.0.0-20180523175556-614d345c47e5
	github.com/appscode/jsonpatch v0.0.0-20190108182946-7c0e3b262f30 // indirect
	github.com/aws/aws-sdk-go v1.38.27
	github.com/chzyer/logex v1.1.10 // indirect
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e // indirect
	github.com/chzyer/test v0.0.0-20210722231415-061457976a23 // indirect
	github.com/coreos/prometheus-operator v0.35.0 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/fatih/color v1.7.0
	github.com/go-logr/zapr v0.1.0 // indirect
	github.com/golang/groupcache v0.0.0-20190129154638-5b532d6fd5ef // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/heroku/docker-registry-client v0.0.0-20181004091502-47ecf50fd8d4
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/juju/ansiterm v0.0.0-20180109212912-720a0952cc2a // indirect
	github.com/lunixbochs/vtclean v1.0.0 // indirect
	github.com/manifoldco/promptui v0.3.1
	github.com/mattn/go-shellwords v1.0.3
	github.com/mitchellh/go-homedir v1.1.0
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/pkg/browser v0.0.0-20180916011732-0a3d74bf9ce4
	github.com/pkg/errors v0.9.1
	github.com/shurcooL/httpfs v0.0.0-20181222201310-74dc9339e414
	github.com/spf13/cobra v0.0.5
	github.com/zalando-incubator/postgres-operator v1.0.1-0.20190129204213-44895939f33f // indirect
	go.uber.org/atomic v1.3.2 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.9.1 // indirect
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.0.0-20190918155943-95b840bb6a1f
	k8s.io/apimachinery v0.0.0-20190913080033-27d36303b655
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.1.12
	sigs.k8s.io/testing_frameworks v0.1.2 // indirect
)

//replace gopkg.in/fsnotify.v1 v1.4.7 => github.com/fsnotify/fsnotify v1.4.7
