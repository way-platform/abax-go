# Getting started with ABAX Open API

## Introduction

The ABAX Open API enables access to information about equipment, vehicles, trips and organization on behalf of ABAX User or ABAX Customer organization.

## Setting up API credentials

In order to authenticate your app in the Authorization Server, you will first need to create API credentials in the ABAX Developer Portal.

Using the ABAX Developer portal you can generate `client_id` and `client_secret` that are required in the authentication flow. You will get API credentials separately for the [sandbox](https://developers.abax.cloud/getting-started#sandbox) environment and the production environment.

Please note, that the name of the credentials will be displayed to end users on the login screen in the case of interactive login. Pick a name, that will make it easy for the users to identify your application.

When creating the credentials, you also need to specify redirect URIs that are allowed to be used in the calls in the OpenID Connect Authorization Code Flow.

## Authorizing access to API

#### General information

Depending on the authorization flow, the access to the API happens in the context of ABAX User or ABAX Customer organization:

- For **public integration** scenarios, where the scope of the data requested is not limited to single ABAX Customer organization, we only allow [Authorization Code Flow](https://developers.abax.cloud/getting-started#authorization-code-flow) with interactive user login in a browser. The user grants access for the integration to use the data on their behalf. The app redirects the user to ABAX Authorization Server, where the user logs in and consents to making the data available for the app.

> **Note:** Currently, the ABAX Open API only supports the context of an Administrator ABAX users. Users with other roles will not be able to use the API.

- For **private integration** scenarios, where the scope of the data requested is limited to single ABAX Customer organization, we also enable [Client Credentials Flow](https://developers.abax.cloud/getting-started#client-credentials-flow) (machine to machine integration). In this case, the authentication is based on the API credentials and the data is scoped to single organization.

> **Note:** By default, API credentials created in the ABAX Developer Portal are configured to use Authorization Code Flow. If you want to use Client Credentials Flow instead, please [contact us](mailto:api@abax.no).

#### Protocol

The authentication and authorization is based on [OpenID Connect protocol](https://openid.net/specs/openid-connect-core-1_0.html) and [OAuth2 protocol](https://tools.ietf.org/html/rfc6749) (M2M). Depending on your platform of choice, you may be able to use a library or component that already implements selected protocol and enables you to obtain an access token required to authorize API calls. You can also implement it on your own. See [Authentication and authorization details](https://developers.abax.cloud/getting-started#authentication-and-authorization-details) and [Samples](https://developers.abax.cloud/getting-started#samples).

> **Note:** If you are building a mobile app, it is strongly advised to implement additional security measures:
>
> - Use the [PKCE](https://developers.abax.cloud/getting-started#pkce)
> - Open the user sign in page in a system browser, instead of embedded webview.

#### Authorization server

The authorization server is available at:

```
https://identity.abax.cloud
```

It supports OpenID Connect Discovery protocol:

```
https://identity.abax.cloud/.well-known/openid-configuration
```

#### Scopes

##### Production specific scopes

- `open_api` - request access to the production ABAX Open API
- `open_api.equipment` - request access to equipment
- `open_api.vehicles` - request access to vehicles
- `open_api.trips` - request access to trips
- `open_api.driving_behaviour` - request access to driving behaviour
- `open_api.organization` - request access to organization

##### Sandbox specific scopes

- `open_api.sandbox` - request access to the ABAX Open API sandbox
- `open_api.sandbox.equipment` - request access to equipment on sandbox environment
- `open_api.sandbox.vehicles` - request access to vehicles on sandbox environment
- `open_api.sandbox.trips` - request access to trips on sandbox environment
- `open_api.sandbox.driving_behaviour` - request access to driving behaviour on sandbox environment
- `open_api.sandbox.organization` - request access to organization on sandbox environment

##### User-related scopes (Authorization Code Flow only)

- `openid` - request the JWT `id_token`
- `abax_profile` - request the `id_token` to include information about the user (name, email and organization id among others)
- `offline_access` - request the refresh token

For example, in order to call the `/v1/vehicles` endpoint, you will have to request at least following scopes:

- `openid`, `abax_profile`, `open_api`, `open_api.vehicles` for Authorization Code Flow
- `open_api`, `open_api.vehicles` for Client Credentials Flow

#### API Capabilities

Access to data is restricted depending on current user's permissions. You can use API Capabilities endpoint to easily check which API operation can be performed by the user.

##### Capabilities list

|Capability|Related endpoints|Comment|
|---|---|---|
|query-equipment|/v1/equipment  <br>/v1/equipment/{id}||
|query-equipment-locations|/v1/equipment/locations||
|query-vehicles-basic|/v1/vehicles  <br>/v1/vehicles/{id}|Access to basic vehicle information|
|query-vehicles|/v1/vehicles  <br>/v1/vehicles/{id}|Access to full vehicle information|
|query-vehicle-locations|/v1/vehicles  <br>/v1/vehicles/{id}  <br>/v1/vehicles/locations|Access to vehicle location data|
|query-vehicle-location-history|/v1/vehicles/{id}/location-history||
|query-vehicle-drive-states|/v1/vehicles/drive-states||
|query-trips|/v1/trips||
|query-trip-route|/v1/trips/{id}/route||
|query-trip-expense-and-extra|/v1/trips/expense||
|query-people|/v1/people  <br>/v1/people/{id}||
|organization-driving-behaviour|/v1/driving-behaviour/organization|Gets organization driving behaviour score.|
|drivers-driving-behaviour|/v1/driving-behaviour/drivers|Gets driving behaviour scores for all the drivers the admin has access to.|
|driver-driving-behaviour|/v1/driving-behaviour/drivers/{id}|Gets driver driving behaviour score by id.|

## Calling the API

Call the selected API endpoint with the access token in the `Authorization` header:

```
GET https://api.abax.cloud/v1/vehicles
Authorization: Bearer [access token]
```

## Samples

[Our samples on GitHub](https://github.com/abax-as/abax-api-samples) demonstrate authentication and calling the API.

## Sandbox

Sandbox environment allows you to test the API interaction without knowing username and password of an actual ABAX user. It also enables you to work on your app even before ABAX grants you access to the production API.

When you use sandbox API credentials in the authorization flow, you will be allowed to log in with a test ABAX User:

```
Username: testadmin
Password: P@ssw0rd
```

> **Note:** when using sandbox, you have to request sandbox scopes. `open_api.sandbox` scope instead of the production `open_api` scope, `open_api.sandbox.trips` scope instead of the production `open_api.trips` etc.

The sandbox ABAX Open API is located at:

```
https://api-test.abax.cloud
```

> **Note:** The sandbox API does not return real data. It returns static, mocked data on each endpoint. If you need more functionality from the sandbox, please [contact us](mailto:api@abax.no).

## Rate limiting and quota

ABAX applies rate limiting to ensure stable operation. When you exceed the quota, the API will be returning `HTTP 429 Too Many Requests` code until the next quota period starts. You can check the current quota usage in the response headers: `X-Ratelimit-Limit-[period]` and `X-Ratelimit-Remaining-[period]`.

The default quota per developer account is 60 requests in 60 seconds. If this is not enough for you, please [contact us](mailto:api@abax.no).

## Authentication and authorization details

> **Note:** The examples below assume using the production environment.

### Authorization Code Flow

Below we present a simplified description of the default Authorization Code Flow. If you need more information, please refer to the [OpenID Connect specification](https://openid.net/specs/openid-connect-core-1_0.html).

In order to authenticate the user and obtain the access token, redirect the user's browser to following URL:

```
https://identity.abax.cloud/connect/authorize?
    response_type=code&
    scope=openid+abax_profile+open_api+open_api.vehicles+offline_access&
    client_id=[client id]&
    redirect_uri=[your redirect uri]
```

> **Note:** Redirect URI has to be on the list of allowed URIs configured for the app.
>
> **Note:** If you are developing a native app, this step requires you to open the authorization server URL in a system browser. The redirect URI should contain a custom scheme your app is registered to, so that you can receive and parse the query parameters added to the URI by authorization server.

The user will then be presented with sign in screen, where they can grant access to the requested scopes. For more information about scopes, see [Scopes](https://developers.abax.cloud/getting-started#scopes).

After successful sign in, user's browser will be redirected to specified redirect URI. Among query parameters, you'll find the `code`, that you can use to get the access token (and optionally the refresh token).

Sample redirect after signin:

```
https://my.app.com?code=[code]&...
```

To obtain the tokens, call:

```
POST https://identity.abax.cloud/connect/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code&code=[code]&client_id=[client id]&client_secret=[client_secret]&redirect_uri=[your redirect uri]
```

> **Note:** The Redirect URI has to be exactly the same as the one specified in authorize endpoint redirect. In case of this request, it serves as an extra security measure.

The JSON response will contain:

- access token
- id token - if `openid` scope was requested
- refresh token - if `offline_access` scope was requested

```json
{
  "id_token": "JWT id token",
  "access_token": "JWT access token",
  "expires_in": 3600,
  "token_type": "Bearer",
  "refresh_token": "refresh token"
}
```

#### Refreshing the access token

By default, the access tokens expire after 60 minutes. In order continue working with the API, new access token has to be issued. This can be done by either repeating the authorization code flow or by using the refresh token. The refresh token itself will expire after 30 days (you need to repeat the authorization code flow in this case).

The refresh token is issued during the authorization process if `offline_access` scope was requested. It can then be used to request new tokens:

```
POST https://identity.abax.cloud/connect/token
Content-Type: application/x-www-form-urlencoded

grant_type=refresh_token&refresh_token=[refresh token]]&client_id=[client key]&client_secret=[client secret]
```

The JSON response will contain new access and refresh tokens.

```json
{
  "id_token": "JWT id token",
  "access_token": "JWT access token",
  "expires_in": 3600,
  "token_type": "Bearer",
  "refresh_token": "refresh token"
}
```

> **Note:** Refresh token can only be used once. When refreshing the access token next time, use the new refresh token returned on the last refresh.

#### PKCE

Mobile app developers are advised to take extra security measure in form of [Proof Key of Code Exchange](https://tools.ietf.org/html/rfc7636). The aim of this extension to Authorization Code Flow is to secure the authorization code from access by rouge apps installed on the user's device.

You first generate a random _code verifier_. According to specification:

_code_verifier = high-entropy cryptographic random STRING using the unreserved characters [A-Z] / [a-z] / [0-9] / "-" / "." / "_" / "~" from Section 2.3 of [RFC3986], with a minimum length of 43 characters and a maximum length of 128 characters._

Example: `6coIVCUqOXwOe1c3dre5RSnw35q1ToHdEJ89dxj2BJR`

Then, you calculate the _code challenge_:

1. Calculate SHA256 hash of code verifier - the result is an array of bytes.
2. Base64-URL encode the bytes, remove padding with `=`.

Example: `iuOX3YMDbdXw1g7GsrBfBUV8G5EA_p18IzcjV-VkcD0`

The code challenge is then used in the authorization URL when user browser gets redirected to sign in screen

```
https://identity.abax.cloud/connect/authorize?
    response_type=code&
    scope=openid+abax_profile+open_api+open_api.vehicles+offline_access&
    client_id=[client id]&
    redirect_uri=[your redirect uri]&
    code_challenge=[code challenge]&
    code_challenge_method=S256
```

When calling the token endpoint, you then send the code verifier in the request and the server validates it against previously presented challenge.

```
POST https://identity.abax.cloud/connect/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code&code=[code]&client_id=[client id]&client_secret=[client secret]&code_verifier=[code verifier]
```

### Client Credentials Flow

This flow is used for private machine to machine integrations, where the data returned from the API is scoped to single ABAX Customer organization. Access token is obtained using pair of `client id` and `client secret`. No user authentication is required. API request is executed in organization context instead of standard user context. If you need more information, please refer to [OAuth2 specification](https://tools.ietf.org/html/rfc6749#section-4.4).

> **Note:** By default, API credentials created in the ABAX Developer Portal are configured to use Authorization Code Flow. If you want to use Client Credentials Flow instead, please [contact us](mailto:api@abax.no).

#### Obtaining Access token

To obtain access token, send HTTP POST request to `/connect/token` endpoint with pair of `client id`, `client secret` and `client_credentials` as grant type.

Sample request:

```
POST https://identity.abax.cloud/connect/token
Content-Type: application/x-www-form-urlencoded

    grant_type=client_credentials&
    scope=open_api.sandbox+open_api.sandbox.vehicles&
    client_id=[client id]&
    client_secret=[client secret]
```

The response will contain JSON with JWT access token, expiration time and list of scopes.

```
{
  "access_token": "JWT access token",
  "expires_in": 3600,
  "token_type": "Bearer",
  "scope": "open_api.sandbox open_api.sandbox.vehicles"
}
```

#### Token expiration

Access token has an expiration time (60 minutes in the above example). After the token expires, a new one has to be requested from the Authorization Server.
