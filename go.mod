module gitlab.com/elixxir/server

go 1.13

require (
	github.com/cznic/mathutil v0.0.0-20181122101859-297441e03548
	github.com/golang/protobuf v1.4.3
	github.com/gopherjs/gopherjs v0.0.0-20200217142428-fce0ec30dd00 // indirect
	github.com/jinzhu/copier v0.0.0-20201025035756-632e723a6687
	github.com/magiconair/properties v1.8.4 // indirect
	github.com/mitchellh/mapstructure v1.4.0 // indirect
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/remyoudompheng/bigfft v0.0.0-20200410134404-eec4a21b6bb0 // indirect
	github.com/smartystreets/assertions v1.2.0 // indirect
	github.com/spf13/afero v1.5.1 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/viper v1.7.1
	gitlab.com/elixxir/comms v0.0.4-0.20210517211849-c6cbb2b9bddd
	gitlab.com/elixxir/crypto v0.0.7-0.20210517212824-4df0efda837f
	gitlab.com/elixxir/gpumathsgo v0.1.0
	gitlab.com/elixxir/primitives v0.0.3-0.20210517205719-63f209b4255b
	gitlab.com/xx_network/comms v0.0.4-0.20210517205649-06ddfa8d2a75
	gitlab.com/xx_network/crypto v0.0.5-0.20210517205543-4ae99cbb9063
	gitlab.com/xx_network/primitives v0.0.4-0.20210517202253-c7b4bd0087ea
	gitlab.com/xx_network/ring v0.0.3-0.20201120004140-b0e268db06d1 // indirect
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b // indirect
	google.golang.org/genproto v0.0.0-20210105202744-fe13368bc0e1 // indirect
	google.golang.org/grpc v1.34.0
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	gorm.io/driver/postgres v1.0.8
	gorm.io/gorm v1.21.7
)

replace google.golang.org/grpc => github.com/grpc/grpc-go v1.27.1
