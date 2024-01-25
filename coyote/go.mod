module coyote

go 1.21.5

replace nhs.uk/coyotecore => ../coyotecore

replace nhs.uk/coyoteadapters => ../coyoteadapters

require (
	nhs.uk/coyoteadapters v0.0.0-00010101000000-000000000000
	nhs.uk/coyotecore v0.0.0-00010101000000-000000000000
)

require (
	github.com/google/go-github/v58 v58.0.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/pelletier/go-toml/v2 v2.1.1 // indirect
)
