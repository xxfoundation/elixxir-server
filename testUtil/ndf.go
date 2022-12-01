////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package testUtil

import (
	"gitlab.com/xx_network/primitives/ndf"
)

var (
	ExampleJSON = `{
	"Timestamp": "2019-06-04T20:48:48-07:00",
	"gateways": [
		{
			"Id": [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1],
			"Address": "52.25.135.52",
			"Tls_certificate": "-----BEGIN CERTIFICATE-----\nMIIDgTCCAmmgAwIBAgIJAKLdZ8UigIAeMA0GCSqGSIb3DQEBBQUAMG8xCzAJBgNV\nBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQx\nGzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjEaMBgGA1UEAwwRZ2F0ZXdheSou\nY21peC5yaXAwHhcNMTkwMzA1MTgzNTU0WhcNMjkwMzAyMTgzNTU0WjBvMQswCQYD\nVQQGEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTESMBAGA1UEBwwJQ2xhcmVtb250\nMRswGQYDVQQKDBJQcml2YXRlZ3JpdHkgQ29ycC4xGjAYBgNVBAMMEWdhdGV3YXkq\nLmNtaXgucmlwMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA9+AaxwDP\nxHbhLmn4HoZu0oUM48Qufc6T5XEZTrpMrqJAouXk+61Jc0EFH96/sbj7VyvnXPRo\ngIENbk2Y84BkB9SkRMIXya/gh9dOEDSgnvj/yg24l3bdKFqBMKiFg00PYB30fU+A\nbe3OI/le0I+v++RwH2AV0BMq+T6PcAGjCC1Q1ZB0wP9/VqNMWq5lbK9wD46IQiSi\n+SgIQeE7HoiAZXrGO0Y7l9P3+VRoXjRQbqfn3ETNL9ZvQuarwAYC9Ix5MxUrS5ag\nOmfjc8bfkpYDFAXRXmdKNISJmtCebX2kDrpP8Bdasx7Fzsx59cEUHCl2aJOWXc7R\n5m3juOVL1HUxjQIDAQABoyAwHjAcBgNVHREEFTATghFnYXRld2F5Ki5jbWl4LnJp\ncDANBgkqhkiG9w0BAQUFAAOCAQEAMu3xoc2LW2UExAAIYYWEETggLNrlGonxteSu\njuJjOR+ik5SVLn0lEu22+z+FCA7gSk9FkWu+v9qnfOfm2Am+WKYWv3dJ5RypW/hD\nNXkOYxVJNYFxeShnHohNqq4eDKpdqSxEcuErFXJdLbZP1uNs4WIOKnThgzhkpuy7\ntZRosvOF1X5uL1frVJzHN5jASEDAa7hJNmQ24kh+ds/Ge39fGD8pK31CWhnIXeDo\nvKD7wivi/gSOBtcRWWLvU8SizZkS3hgTw0lSOf5geuzvasCEYlqrKFssj6cTzbCB\nxy3ra3WazRTNTW4TmkHlCUC9I3oWTTxw5iQxF/I2kQQnwR7L3w==\n-----END CERTIFICATE-----"
		},
		{
			"Id": [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1],
			"Address": "52.25.219.38",
			"Tls_certificate": "-----BEGIN CERTIFICATE-----\nMIIDgTCCAmmgAwIBAgIJAKLdZ8UigIAeMA0GCSqGSIb3DQEBBQUAMG8xCzAJBgNV\nBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQx\nGzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjEaMBgGA1UEAwwRZ2F0ZXdheSou\nY21peC5yaXAwHhcNMTkwMzA1MTgzNTU0WhcNMjkwMzAyMTgzNTU0WjBvMQswCQYD\nVQQGEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTESMBAGA1UEBwwJQ2xhcmVtb250\nMRswGQYDVQQKDBJQcml2YXRlZ3JpdHkgQ29ycC4xGjAYBgNVBAMMEWdhdGV3YXkq\nLmNtaXgucmlwMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA9+AaxwDP\nxHbhLmn4HoZu0oUM48Qufc6T5XEZTrpMrqJAouXk+61Jc0EFH96/sbj7VyvnXPRo\ngIENbk2Y84BkB9SkRMIXya/gh9dOEDSgnvj/yg24l3bdKFqBMKiFg00PYB30fU+A\nbe3OI/le0I+v++RwH2AV0BMq+T6PcAGjCC1Q1ZB0wP9/VqNMWq5lbK9wD46IQiSi\n+SgIQeE7HoiAZXrGO0Y7l9P3+VRoXjRQbqfn3ETNL9ZvQuarwAYC9Ix5MxUrS5ag\nOmfjc8bfkpYDFAXRXmdKNISJmtCebX2kDrpP8Bdasx7Fzsx59cEUHCl2aJOWXc7R\n5m3juOVL1HUxjQIDAQABoyAwHjAcBgNVHREEFTATghFnYXRld2F5Ki5jbWl4LnJp\ncDANBgkqhkiG9w0BAQUFAAOCAQEAMu3xoc2LW2UExAAIYYWEETggLNrlGonxteSu\njuJjOR+ik5SVLn0lEu22+z+FCA7gSk9FkWu+v9qnfOfm2Am+WKYWv3dJ5RypW/hD\nNXkOYxVJNYFxeShnHohNqq4eDKpdqSxEcuErFXJdLbZP1uNs4WIOKnThgzhkpuy7\ntZRosvOF1X5uL1frVJzHN5jASEDAa7hJNmQ24kh+ds/Ge39fGD8pK31CWhnIXeDo\nvKD7wivi/gSOBtcRWWLvU8SizZkS3hgTw0lSOf5geuzvasCEYlqrKFssj6cTzbCB\nxy3ra3WazRTNTW4TmkHlCUC9I3oWTTxw5iQxF/I2kQQnwR7L3w==\n-----END CERTIFICATE-----"
		},
		{
			"Id": [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1],
			"Address": "52.41.80.104",
			"Tls_certificate": "-----BEGIN CERTIFICATE-----\nMIIDgTCCAmmgAwIBAgIJAKLdZ8UigIAeMA0GCSqGSIb3DQEBBQUAMG8xCzAJBgNV\nBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQx\nGzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjEaMBgGA1UEAwwRZ2F0ZXdheSou\nY21peC5yaXAwHhcNMTkwMzA1MTgzNTU0WhcNMjkwMzAyMTgzNTU0WjBvMQswCQYD\nVQQGEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTESMBAGA1UEBwwJQ2xhcmVtb250\nMRswGQYDVQQKDBJQcml2YXRlZ3JpdHkgQ29ycC4xGjAYBgNVBAMMEWdhdGV3YXkq\nLmNtaXgucmlwMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA9+AaxwDP\nxHbhLmn4HoZu0oUM48Qufc6T5XEZTrpMrqJAouXk+61Jc0EFH96/sbj7VyvnXPRo\ngIENbk2Y84BkB9SkRMIXya/gh9dOEDSgnvj/yg24l3bdKFqBMKiFg00PYB30fU+A\nbe3OI/le0I+v++RwH2AV0BMq+T6PcAGjCC1Q1ZB0wP9/VqNMWq5lbK9wD46IQiSi\n+SgIQeE7HoiAZXrGO0Y7l9P3+VRoXjRQbqfn3ETNL9ZvQuarwAYC9Ix5MxUrS5ag\nOmfjc8bfkpYDFAXRXmdKNISJmtCebX2kDrpP8Bdasx7Fzsx59cEUHCl2aJOWXc7R\n5m3juOVL1HUxjQIDAQABoyAwHjAcBgNVHREEFTATghFnYXRld2F5Ki5jbWl4LnJp\ncDANBgkqhkiG9w0BAQUFAAOCAQEAMu3xoc2LW2UExAAIYYWEETggLNrlGonxteSu\njuJjOR+ik5SVLn0lEu22+z+FCA7gSk9FkWu+v9qnfOfm2Am+WKYWv3dJ5RypW/hD\nNXkOYxVJNYFxeShnHohNqq4eDKpdqSxEcuErFXJdLbZP1uNs4WIOKnThgzhkpuy7\ntZRosvOF1X5uL1frVJzHN5jASEDAa7hJNmQ24kh+ds/Ge39fGD8pK31CWhnIXeDo\nvKD7wivi/gSOBtcRWWLvU8SizZkS3hgTw0lSOf5geuzvasCEYlqrKFssj6cTzbCB\nxy3ra3WazRTNTW4TmkHlCUC9I3oWTTxw5iQxF/I2kQQnwR7L3w==\n-----END CERTIFICATE-----"
		}
	],
	"nodes": [
		{
			"Id": [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2],
			"Address": "18.237.147.105",
			"Tls_certificate": "-----BEGIN CERTIFICATE-----\nMIIDbDCCAlSgAwIBAgIJAOUNtZneIYECMA0GCSqGSIb3DQEBBQUAMGgxCzAJBgNV\nBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQx\nGzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjETMBEGA1UEAwwKKi5jbWl4LnJp\ncDAeFw0xOTAzMDUxODM1NDNaFw0yOTAzMDIxODM1NDNaMGgxCzAJBgNVBAYTAlVT\nMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQxGzAZBgNV\nBAoMElByaXZhdGVncml0eSBDb3JwLjETMBEGA1UEAwwKKi5jbWl4LnJpcDCCASIw\nDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAPP0WyVkfZA/CEd2DgKpcudn0oDh\nDwsjmx8LBDWsUgQzyLrFiVigfUmUefknUH3dTJjmiJtGqLsayCnWdqWLHPJYvFfs\nWYW0IGF93UG/4N5UAWO4okC3CYgKSi4ekpfw2zgZq0gmbzTnXcHF9gfmQ7jJUKSE\ntJPSNzXq+PZeJTC9zJAb4Lj8QzH18rDM8DaL2y1ns0Y2Hu0edBFn/OqavBJKb/uA\nm3AEjqeOhC7EQUjVamWlTBPt40+B/6aFJX5BYm2JFkRsGBIyBVL46MvC02MgzTT9\nbJIJfwqmBaTruwemNgzGu7Jk03hqqS1TUEvSI6/x8bVoba3orcKkf9HsDjECAwEA\nAaMZMBcwFQYDVR0RBA4wDIIKKi5jbWl4LnJpcDANBgkqhkiG9w0BAQUFAAOCAQEA\nneUocN4AbcQAC1+b3To8u5UGdaGxhcGyZBlAoenRVdjXK3lTjsMdMWb4QctgNfIf\nU/zuUn2mxTmF/ekP0gCCgtleZr9+DYKU5hlXk8K10uKxGD6EvoiXZzlfeUuotgp2\nqvI3ysOm/hvCfyEkqhfHtbxjV7j7v7eQFPbvNaXbLa0yr4C4vMK/Z09Ui9JrZ/Z4\ncyIkxfC6/rOqAirSdIp09EGiw7GM8guHyggE4IiZrDslT8V3xIl985cbCxSxeW1R\ntgH4rdEXuVe9+31oJhmXOE9ux2jCop9tEJMgWg7HStrJ5plPbb+HmjoX3nBO04E5\n6m52PyzMNV+2N21IPppKwA==\n-----END CERTIFICATE-----"
		},
		{
			"Id": [1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2],
			"Address": "52.11.136.238",
			"Tls_certificate": "-----BEGIN CERTIFICATE-----\nMIIDbDCCAlSgAwIBAgIJAOUNtZneIYECMA0GCSqGSIb3DQEBBQUAMGgxCzAJBgNV\nBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQx\nGzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjETMBEGA1UEAwwKKi5jbWl4LnJp\ncDAeFw0xOTAzMDUxODM1NDNaFw0yOTAzMDIxODM1NDNaMGgxCzAJBgNVBAYTAlVT\nMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQxGzAZBgNV\nBAoMElByaXZhdGVncml0eSBDb3JwLjETMBEGA1UEAwwKKi5jbWl4LnJpcDCCASIw\nDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAPP0WyVkfZA/CEd2DgKpcudn0oDh\nDwsjmx8LBDWsUgQzyLrFiVigfUmUefknUH3dTJjmiJtGqLsayCnWdqWLHPJYvFfs\nWYW0IGF93UG/4N5UAWO4okC3CYgKSi4ekpfw2zgZq0gmbzTnXcHF9gfmQ7jJUKSE\ntJPSNzXq+PZeJTC9zJAb4Lj8QzH18rDM8DaL2y1ns0Y2Hu0edBFn/OqavBJKb/uA\nm3AEjqeOhC7EQUjVamWlTBPt40+B/6aFJX5BYm2JFkRsGBIyBVL46MvC02MgzTT9\nbJIJfwqmBaTruwemNgzGu7Jk03hqqS1TUEvSI6/x8bVoba3orcKkf9HsDjECAwEA\nAaMZMBcwFQYDVR0RBA4wDIIKKi5jbWl4LnJpcDANBgkqhkiG9w0BAQUFAAOCAQEA\nneUocN4AbcQAC1+b3To8u5UGdaGxhcGyZBlAoenRVdjXK3lTjsMdMWb4QctgNfIf\nU/zuUn2mxTmF/ekP0gCCgtleZr9+DYKU5hlXk8K10uKxGD6EvoiXZzlfeUuotgp2\nqvI3ysOm/hvCfyEkqhfHtbxjV7j7v7eQFPbvNaXbLa0yr4C4vMK/Z09Ui9JrZ/Z4\ncyIkxfC6/rOqAirSdIp09EGiw7GM8guHyggE4IiZrDslT8V3xIl985cbCxSxeW1R\ntgH4rdEXuVe9+31oJhmXOE9ux2jCop9tEJMgWg7HStrJ5plPbb+HmjoX3nBO04E5\n6m52PyzMNV+2N21IPppKwA==\n-----END CERTIFICATE-----"
		},
		{
			"Id": [2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2],
			"Address": "34.213.79.31",
			"Tls_certificate": "-----BEGIN CERTIFICATE-----\nMIIDbDCCAlSgAwIBAgIJAOUNtZneIYECMA0GCSqGSIb3DQEBBQUAMGgxCzAJBgNV\nBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQx\nGzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjETMBEGA1UEAwwKKi5jbWl4LnJp\ncDAeFw0xOTAzMDUxODM1NDNaFw0yOTAzMDIxODM1NDNaMGgxCzAJBgNVBAYTAlVT\nMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQxGzAZBgNV\nBAoMElByaXZhdGVncml0eSBDb3JwLjETMBEGA1UEAwwKKi5jbWl4LnJpcDCCASIw\nDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAPP0WyVkfZA/CEd2DgKpcudn0oDh\nDwsjmx8LBDWsUgQzyLrFiVigfUmUefknUH3dTJjmiJtGqLsayCnWdqWLHPJYvFfs\nWYW0IGF93UG/4N5UAWO4okC3CYgKSi4ekpfw2zgZq0gmbzTnXcHF9gfmQ7jJUKSE\ntJPSNzXq+PZeJTC9zJAb4Lj8QzH18rDM8DaL2y1ns0Y2Hu0edBFn/OqavBJKb/uA\nm3AEjqeOhC7EQUjVamWlTBPt40+B/6aFJX5BYm2JFkRsGBIyBVL46MvC02MgzTT9\nbJIJfwqmBaTruwemNgzGu7Jk03hqqS1TUEvSI6/x8bVoba3orcKkf9HsDjECAwEA\nAaMZMBcwFQYDVR0RBA4wDIIKKi5jbWl4LnJpcDANBgkqhkiG9w0BAQUFAAOCAQEA\nneUocN4AbcQAC1+b3To8u5UGdaGxhcGyZBlAoenRVdjXK3lTjsMdMWb4QctgNfIf\nU/zuUn2mxTmF/ekP0gCCgtleZr9+DYKU5hlXk8K10uKxGD6EvoiXZzlfeUuotgp2\nqvI3ysOm/hvCfyEkqhfHtbxjV7j7v7eQFPbvNaXbLa0yr4C4vMK/Z09Ui9JrZ/Z4\ncyIkxfC6/rOqAirSdIp09EGiw7GM8guHyggE4IiZrDslT8V3xIl985cbCxSxeW1R\ntgH4rdEXuVe9+31oJhmXOE9ux2jCop9tEJMgWg7HStrJ5plPbb+HmjoX3nBO04E5\n6m52PyzMNV+2N21IPppKwA==\n-----END CERTIFICATE-----"
		}
	],
	"registration": {
		"Address": "registration.default.cmix.rip",
		"Tls_certificate": "-----BEGIN CERTIFICATE-----\nMIIDkDCCAnigAwIBAgIJAJnjosuSsP7gMA0GCSqGSIb3DQEBBQUAMHQxCzAJBgNV\nBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQx\nGzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjEfMB0GA1UEAwwWcmVnaXN0cmF0\naW9uKi5jbWl4LnJpcDAeFw0xOTAzMDUyMTQ5NTZaFw0yOTAzMDIyMTQ5NTZaMHQx\nCzAJBgNVBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFy\nZW1vbnQxGzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjEfMB0GA1UEAwwWcmVn\naXN0cmF0aW9uKi5jbWl4LnJpcDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoC\nggEBAOQKvqjdh35o+MECBhCwopJzPlQNmq2iPbewRNtI02bUNK3kLQUbFlYdzNGZ\nS4GYXGc5O+jdi8Slx82r1kdjz5PPCNFBARIsOP/L8r3DGeW+yeJdgBZjm1s3ylka\nmt4Ajiq/bNjysS6L/WSOp+sVumDxtBEzO/UTU1O6QRnzUphLaiWENmErGvsH0CZV\nq38Ia58k/QjCAzpUcYi4j2l1fb07xqFcQD8H6SmUM297UyQosDrp8ukdIo31Koxr\n4XDnnNNsYStC26tzHMeKuJ2Wl+3YzsSyflfM2YEcKE31sqB9DS36UkJ8J84eLsHN\nImGg3WodFAviDB67+jXDbB30NkMCAwEAAaMlMCMwIQYDVR0RBBowGIIWcmVnaXN0\ncmF0aW9uKi5jbWl4LnJpcDANBgkqhkiG9w0BAQUFAAOCAQEARU2Fkf6QbEYPr3hZ\nOn+v3uRgcVY0M+VW79F/j6sreGCaAWaJRxlKQFCXc24/hKuRKPXAyPOLzoPHQpvD\nwBwdYfzzEVrVQGE1wOasyj/4bKcqAoZE+t3RzfyQyPF826IEQ5BKM4vHhX7TE3o4\n+F7PR9l0U/3Zb2dIa6gdnj7OPp6wFGHQbNYpsfFZEmowiWrptYy8U8mGfLitmgpG\n3YBKjmkVtuTfVKqJKaz5V1GKUbBzcRxAFdXzzapIRP72P1S/oaI6z3FqhVSY+JYQ\nJ8nY3jklZ2C6fuufZyfAfhHYj/YUqfbqKpN28MqLB5AnPeIwQvi7zCuVEYMq6oyE\n6qSZnA==\n-----END CERTIFICATE-----"
	},
	"notification": {
		"Address": "notification.default.cmix.rip",
		"Tls_certificate": "-----BEGIN CERTIFICATE-----\nMIIDkDCCAnigAwIBAgIJAJnjosuSsP7gMA0GCSqGSIb3DQEBBQUAMHQxCzAJBgNV\nBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQx\nGzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjEfMB0GA1UEAwwWcmVnaXN0cmF0\naW9uKi5jbWl4LnJpcDAeFw0xOTAzMDUyMTQ5NTZaFw0yOTAzMDIyMTQ5NTZaMHQx\nCzAJBgNVBAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFy\nZW1vbnQxGzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjEfMB0GA1UEAwwWcmVn\naXN0cmF0aW9uKi5jbWl4LnJpcDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoC\nggEBAOQKvqjdh35o+MECBhCwopJzPlQNmq2iPbewRNtI02bUNK3kLQUbFlYdzNGZ\nS4GYXGc5O+jdi8Slx82r1kdjz5PPCNFBARIsOP/L8r3DGeW+yeJdgBZjm1s3ylka\nmt4Ajiq/bNjysS6L/WSOp+sVumDxtBEzO/UTU1O6QRnzUphLaiWENmErGvsH0CZV\nq38Ia58k/QjCAzpUcYi4j2l1fb07xqFcQD8H6SmUM297UyQosDrp8ukdIo31Koxr\n4XDnnNNsYStC26tzHMeKuJ2Wl+3YzsSyflfM2YEcKE31sqB9DS36UkJ8J84eLsHN\nImGg3WodFAviDB67+jXDbB30NkMCAwEAAaMlMCMwIQYDVR0RBBowGIIWcmVnaXN0\ncmF0aW9uKi5jbWl4LnJpcDANBgkqhkiG9w0BAQUFAAOCAQEAF9mNzk+g+o626Rll\nt3f3/1qIyYQrYJ0BjSWCKYEFMCgZ4JibAJjAvIajhVYERtltffM+YKcdE2kTpdzJ\n0YJuUnRfuv6sVnXlVVugUUnd4IOigmjbCdM32k170CYMm0aiwGxl4FrNa8ei7AIa\nx/s1n+sqWq3HeW5LXjnoVb+s3HeCWIuLfcgrurfye8FnNhy14HFzxVYYefIKm0XL\n+DPlcGGGm/PPYt3u4a2+rP3xaihc65dTa0u5tf/XPXtPxTDPFj2JeQDFxo7QRREb\nPD89CtYnwuP937CrkvCKrL0GkW1FViXKqZY9F5uhxrvLIpzhbNrs/EbtweY35XGL\nDCCMkg==\n-----END CERTIFICATE-----"
	},
	"udb": {
		"Id": [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 0]
	},
	"E2e": {
		"Prime": "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF",
		"Small_prime": "7FFFFFFFFFFFFFFFE487ED5110B4611A62633145C06E0E68948127044533E63A0105DF531D89CD9128A5043CC71A026EF7CA8CD9E69D218D98158536F92F8A1BA7F09AB6B6A8E122F242DABB312F3F637A262174D31BF6B585FFAE5B7A035BF6F71C35FDAD44CFD2D74F9208BE258FF324943328F6722D9EE1003E5C50B1DF82CC6D241B0E2AE9CD348B1FD47E9267AFC1B2AE91EE51D6CB0E3179AB1042A95DCF6A9483B84B4B36B3861AA7255E4C0278BA3604650C10BE19482F23171B671DF1CF3B960C074301CD93C1D17603D147DAE2AEF837A62964EF15E5FB4AAC0B8C1CCAA4BE754AB5728AE9130C4C7D02880AB9472D455655347FFFFFFFFFFFFFFF",
		"Generator": "02"
	},
	"Cmix": {
		"Prime": "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF",
		"Small_prime": "7FFFFFFFFFFFFFFFE487ED5110B4611A62633145C06E0E68948127044533E63A0105DF531D89CD9128A5043CC71A026EF7CA8CD9E69D218D98158536F92F8A1BA7F09AB6B6A8E122F242DABB312F3F637A262174D31BF6B585FFAE5B7A035BF6F71C35FDAD44CFD2D74F9208BE258FF324943328F6722D9EE1003E5C50B1DF82CC6D241B0E2AE9CD348B1FD47E9267AFC1B2AE91EE51D6CB0E3179AB1042A95DCF6A9483B84B4B36B3861AA7255E4C0278BA3604650C10BE19482F23171B671DF1CF3B960C074301CD93C1D17603D147DAE2AEF837A62964EF15E5FB4AAC0B8C1CCAA4BE754AB5728AE9130C4C7D02880AB9472D455655347FFFFFFFFFFFFFFF",
		"Generator": "02"
	}
}`
	NDF, _     = ndf.Unmarshal(ExampleNDF)
	ExampleNDF = []byte(ExampleJSON)
	RegPubKey  = `-----BEGIN PUBLIC KEY-----
MIICCgKCAgEAq3ycqsFadls+shZZj7RwXV331BNagJFYGSdRnxK2Hzv/cBzGH5Io
Y4m/WRUbDgbCHTRCvT6f51SQB7Hq0yxAMF6bwdRHk18kfzE4fLVY0Ll+kVSIlNoE
yClY2XFLOwWiwS4WF0PF6Y4lEwZbah2taY+4lNeveNYvE8ZEdCjPdGl3ymQaXCXX
9PoXNitRT1WK7TLLhITgN3Dg76SFOI0XoxUhSiXjcGksPWNMX/6kywc+fnXYksPt
8dwMC7rsE8z1UxnSnue3Tlc8KvDeyk82Ka5LGYAnRBVXhi8tFmoUlWJeyBn80BR1
+Cd6673ec1P1Zy+e3qg6Qe4ee6cpXta6FjU6EQjoEp/Xrkj/iXKvD4NtIut3Yww2
i9NOFONRTU3HJ6UhCJHen++Idi6mOuGNQLTG3aMwFuXehI1gBodLWV3N2h4fKTAa
ifYPzwBXcAkwJ9kPYJiOjc6CLheAxY5GtBsa5Bcqfn9oozmJog8Mf6aTJylOBZ04
puBTVE+oSs3aG+ALyl4TjBBzXgKAN94egL7mHMh2gWjhdqRZTRvt0GTaL8ONIAC7
Htf8nUR1qhqNYOAfW6F16dl6t4HFahEoVTCTSXlyhDh0PA7BZ3AK++4wjfahCAkl
MiAmQ5fylZUUnueIuFSfujaGvUvF5gH8wJbAQHUaIBWyxwLuOYl4qc8CAwEAAQ==
-----END PUBLIC KEY-----`
	RegCert = `-----BEGIN CERTIFICATE-----
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
-----END CERTIFICATE-----
`
	RegPrivKey = `-----BEGIN PRIVATE KEY-----
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
jR+QSAa9eEozCngV6LUagC0YYWDZ
-----END PRIVATE KEY-----
`
)
