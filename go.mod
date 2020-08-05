module gitlab.com/elixxir/server

go 1.13

require (
	github.com/cznic/mathutil v0.0.0-20181122101859-297441e03548
	github.com/go-pg/pg v8.0.6+incompatible
	github.com/golang/protobuf v1.4.2
	github.com/gopherjs/gopherjs v0.0.0-20200217142428-fce0ec30dd00 // indirect
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.3.0 // indirect
	github.com/onsi/ginkgo v1.12.0 // indirect
	github.com/onsi/gomega v1.9.0 // indirect
	github.com/pelletier/go-toml v1.7.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/remyoudompheng/bigfft v0.0.0-20200410134404-eec4a21b6bb0 // indirect
	github.com/smartystreets/assertions v1.1.0 // indirect
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v1.0.0
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.6.3
	gitlab.com/elixxir/comms v0.0.0-20200804225939-84dbe3cccc62
	gitlab.com/elixxir/crypto v0.0.0-20200804231945-1354885c51cd
	gitlab.com/elixxir/gpumathsgo v0.0.2-0.20200617001921-1de1fff56304
	gitlab.com/elixxir/primitives v0.0.0-20200804231232-ad79a9e8f113
	gitlab.com/xx_network/comms v0.0.0-20200804225654-09a9af23d699
	gitlab.com/xx_network/primitives v0.0.0-20200804183002-f99f7a7284da
	golang.org/x/crypto v0.0.0-20200707235045-ab33eee955e0
	google.golang.org/grpc v1.30.0
	gopkg.in/ini.v1 v1.55.0 // indirect
	gopkg.in/yaml.v2 v2.3.0
	mellium.im/sasl v0.0.0-20190815210834-e27ea4901008 // indirect
)

replace google.golang.org/grpc => github.com/grpc/grpc-go v1.27.1
