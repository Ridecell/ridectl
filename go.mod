module github.com/Ridecell/ridectl

go 1.16

require (
	github.com/Ridecell/ridecell-controllers v0.0.0-20211019083602-39744a13278a
	github.com/Ridecell/summon-operator v0.0.0-20220331070632-01baed89215b
	github.com/aws/aws-sdk-go v1.40.29
	github.com/heroku/docker-registry-client v0.0.0-20190909225348-afc9e1acc3d5
	github.com/inconshreveable/go-update v0.0.0-20160112193335-8152e7eb6ccf
	github.com/manifoldco/promptui v0.8.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/pterm/pterm v0.12.31
	github.com/spf13/cobra v1.2.1
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v13.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.9.6
	sigs.k8s.io/yaml v1.2.0
)

replace k8s.io/client-go => k8s.io/client-go v0.21.0

replace github.com/abbot/go-http-auth => github.com/containous/go-http-auth v0.4.1-0.20200324110947-a37a7636d23e

replace github.com/go-check/check => github.com/containous/check v0.0.0-20170915194414-ca0bf163426a
