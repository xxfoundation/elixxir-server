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
	gitlab.com/elixxir/comms v0.0.0-20200925164459-6dde2d05f8ee
	gitlab.com/elixxir/crypto v0.0.0-20201002151041-c4ab8f8033dc
	gitlab.com/elixxir/gpumathsgo v0.0.2-0.20201009190732-5b74fe17407d
	gitlab.com/elixxir/primitives v0.0.0-20200916172343-37503735c7a1
	gitlab.com/xx_network/comms v0.0.0-20200925163950-5686bc0f8ab0
	gitlab.com/xx_network/crypto v0.0.0-20200812183430-c77a5281c686
	gitlab.com/xx_network/primitives v0.0.0-20200812183720-516a65a4a9b2
	golang.org/x/crypto v0.0.0-20200728195943-123391ffb6de
	google.golang.org/grpc v1.31.0
	gopkg.in/ini.v1 v1.55.0 // indirect
	gopkg.in/yaml.v2 v2.3.0
	mellium.im/sasl v0.0.0-20190815210834-e27ea4901008 // indirect
)

replace google.golang.org/grpc => github.com/grpc/grpc-go v1.27.1
