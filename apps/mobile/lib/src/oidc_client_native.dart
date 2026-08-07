import 'package:flutter/foundation.dart';
import 'package:flutter_appauth/flutter_appauth.dart';

import 'calligraphy_api.dart';
import 'models.dart';
import 'oidc_client_contract.dart';

const _configuredClientId = String.fromEnvironment(
  'CALLIGRAPHY_OIDC_CLIENT_ID',
);
const _redirectUri = String.fromEnvironment(
  'CALLIGRAPHY_OIDC_REDIRECT_URI',
  defaultValue: 'com.nebula.calligraphy:/oauthredirect',
);
const _allowInsecureOidc = bool.fromEnvironment(
  'CALLIGRAPHY_ALLOW_INSECURE_OIDC',
);

OidcAuthorizationClient createOidcAuthorizationClient() =>
    const AppAuthOidcAuthorizationClient();

class AppAuthOidcAuthorizationClient implements OidcAuthorizationClient {
  const AppAuthOidcAuthorizationClient({FlutterAppAuth? appAuth})
    : _appAuth = appAuth ?? const FlutterAppAuth();

  final FlutterAppAuth _appAuth;

  @override
  Future<OidcAuthorizationResult> authorize(RuntimeConfig config) async {
    final allowInsecure = _allowInsecureOidc && !kReleaseMode;
    final clientId = _configuredClientId.isNotEmpty
        ? _configuredClientId
        : config.identityClientId;
    if (clientId.isEmpty ||
        config.identityTenant.isEmpty ||
        config.identityAuthorizationEndpoint.isEmpty ||
        config.identityTokenEndpoint.isEmpty) {
      throw const ApiException(503, 'oidc_not_configured', 'OIDC 原生登录配置不完整');
    }
    for (final endpoint in [
      config.identityAuthorizationEndpoint,
      config.identityTokenEndpoint,
    ]) {
      final uri = Uri.tryParse(endpoint);
      if (uri == null ||
          !uri.hasAuthority ||
          (uri.scheme != 'https' && !allowInsecure)) {
        throw const ApiException(
          503,
          'oidc_endpoint_insecure',
          'OIDC 原生登录必须使用 HTTPS',
        );
      }
    }

    final response = await _appAuth.authorizeAndExchangeCode(
      AuthorizationTokenRequest(
        clientId,
        _redirectUri,
        serviceConfiguration: AuthorizationServiceConfiguration(
          authorizationEndpoint: config.identityAuthorizationEndpoint,
          tokenEndpoint: config.identityTokenEndpoint,
        ),
        scopes: const ['openid', 'profile', 'email'],
        additionalParameters: {'tenant': config.identityTenant},
        allowInsecureConnections: allowInsecure,
      ),
    );
    final accessToken = response.accessToken ?? '';
    if (accessToken.isEmpty) {
      throw const ApiException(502, 'oidc_token_missing', 'Identity 未返回访问令牌');
    }
    return OidcAuthorizationResult(accessToken: accessToken);
  }
}
