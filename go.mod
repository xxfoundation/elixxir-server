module gitlab.com/elixxir/server

go 1.13

require (
	github.com/cznic/mathutil v0.0.0-20181122101859-297441e03548
	github.com/go-pg/pg v8.0.7+incompatible
	github.com/golang/protobuf v1.4.3
	github.com/gopherjs/gopherjs v0.0.0-20200217142428-fce0ec30dd00 // indirect
	github.com/jinzhu/copier v0.0.0-20201025035756-632e723a6687
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/magiconair/properties v1.8.4 // indirect
	github.com/mitchellh/mapstructure v1.3.3 // indirect
	github.com/nxadm/tail v1.4.5 // indirect
	github.com/onsi/ginkgo v1.14.2 // indirect
	github.com/onsi/gomega v1.10.3 // indirect
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/remyoudompheng/bigfft v0.0.0-20200410134404-eec4a21b6bb0 // indirect
	github.com/smartystreets/assertions v1.2.0 // indirect
	github.com/spf13/afero v1.4.1 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/viper v1.7.1
	gitlab.com/elixxir/comms v0.0.4-0.20201130220746-47f71717b918
	gitlab.com/elixxir/crypto v0.0.6
	gitlab.com/elixxir/gpumathsgo v0.0.2-0.20201105175137-191e95bf062b
	gitlab.com/elixxir/primitives v0.0.3-0.20201116174806-97f190989704
	gitlab.com/xx_network/comms v0.0.4-0.20201119231004-a67d08045535
	gitlab.com/xx_network/crypto v0.0.5-0.20201124194022-366c10b1bce0
	gitlab.com/xx_network/primitives v0.0.3-0.20201116234927-44e42fc91e7c
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897
	google.golang.org/grpc v1.33.2
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/yaml.v2 v2.3.0
	mellium.im/sasl v0.0.0-20190815210834-e27ea4901008 // indirect
)

replace google.golang.org/grpc => github.com/grpc/grpc-go v1.27.1
