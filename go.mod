module github.com/Ridecell/ridectl

go 1.19

require (
	github.com/Ridecell/ridecell-controllers v0.0.0-20211019083602-39744a13278a
	github.com/Ridecell/summon-operator v0.0.0-20230324100236-d03a8d335e46
	github.com/aws/aws-sdk-go-v2 v1.17.8
	github.com/aws/aws-sdk-go-v2/config v1.18.21
	github.com/aws/aws-sdk-go-v2/service/kms v1.20.10
	github.com/inconshreveable/go-update v0.0.0-20160112193335-8152e7eb6ccf
	github.com/manifoldco/promptui v0.9.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/pterm/pterm v0.12.58
	github.com/spf13/cobra v1.7.0
	golang.org/x/crypto v0.8.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.26.3
	k8s.io/apimachinery v0.26.3
	k8s.io/client-go v13.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.11.2
)

require (
	atomicgo.dev/cursor v0.1.1 // indirect
	atomicgo.dev/keyboard v0.2.9 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.13.20 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.13.2 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.32 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.26 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.33 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.26 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.12.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.14.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.18.9 // indirect
	github.com/aws/smithy-go v1.13.5 // indirect
	github.com/chzyer/readline v1.5.1 // indirect
	github.com/containerd/console v1.0.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/emicklei/go-restful/v3 v3.9.0 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.1 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/gnostic v0.5.7-v3refs // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gookit/color v1.5.3 // indirect
	github.com/imdario/mergo v0.3.15 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/lithammer/fuzzysearch v1.1.5 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/rogpeppe/go-internal v1.8.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/oauth2 v0.7.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/term v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.26.3 // indirect
	k8s.io/klog/v2 v2.90.1 // indirect
	k8s.io/kube-openapi v0.0.0-20230327201221-f5883ff37f0c // indirect
	k8s.io/utils v0.0.0-20230406110748-d93618cff8a2 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace k8s.io/client-go => k8s.io/client-go v0.24.13

replace github.com/abbot/go-http-auth => github.com/containous/go-http-auth v0.4.1-0.20200324110947-a37a7636d23e

replace github.com/go-check/check => github.com/containous/check v0.0.0-20170915194414-ca0bf163426a
