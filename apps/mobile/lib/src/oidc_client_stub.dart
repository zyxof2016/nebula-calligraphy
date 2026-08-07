import 'calligraphy_api.dart';
import 'models.dart';
import 'oidc_client_contract.dart';

OidcAuthorizationClient createOidcAuthorizationClient() =>
    const UnsupportedOidcAuthorizationClient();

class UnsupportedOidcAuthorizationClient implements OidcAuthorizationClient {
  const UnsupportedOidcAuthorizationClient();

  @override
  Future<OidcAuthorizationResult> authorize(RuntimeConfig config) {
    throw const ApiException(
      400,
      'oidc_platform_unsupported',
      '请使用原生客户端或浏览器版完成统一登录',
    );
  }
}
