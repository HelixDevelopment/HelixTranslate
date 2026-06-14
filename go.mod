module digital.vasic.translator

go 1.26

require (
	github.com/gin-gonic/gin v1.11.0
	github.com/golang-jwt/jwt/v5 v5.3.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/lib/pq v1.10.9
	github.com/mattn/go-sqlite3 v1.14.37
	github.com/quic-go/quic-go v0.56.0
	github.com/redis/go-redis/v9 v9.7.0
	github.com/spf13/cobra v1.10.2
	github.com/stretchr/testify v1.11.1
	golang.org/x/crypto v0.49.0
	golang.org/x/net v0.52.0
	golang.org/x/text v0.35.0
	golang.org/x/time v0.14.0
	google.golang.org/grpc v1.80.0
	google.golang.org/protobuf v1.36.11
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/bytedance/sonic v1.14.0 // indirect
	github.com/bytedance/sonic/loader v0.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/gin-contrib/sse v1.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.27.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/goccy/go-yaml v1.18.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/quic-go/qpack v0.5.1 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/stretchr/objx v0.5.3 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.3.0 // indirect
	go.opentelemetry.io/otel v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.43.0 // indirect
	golang.org/x/arch v0.20.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260401024825-9d38bb4040a9 // indirect
)

require (
	digital.vasic.challenges v0.0.0
	digital.vasic.containers v0.0.0-00010101000000-000000000000
	digital.vasic.llmsverifier v0.0.0
	digital.vasic.models v0.0.0-00010101000000-000000000000
	github.com/ledongthuc/pdf v0.0.0-20250511090121-5959a4027728
)

replace digital.vasic.challenges => ./challenges

replace digital.vasic.containers => ./containers

replace digital.vasic.helixqa => ./helix_qa

replace digital.vasic.docprocessor => ./doc_processor

replace digital.vasic.llmorchestrator => ./llm_orchestrator

replace digital.vasic.llmprovider => ./llm_provider

replace digital.vasic.visionengine => ./vision_engine

replace digital.vasic.llmsverifier => ./llms_verifier/llm-verifier

replace digital.vasic.models => ./Models
