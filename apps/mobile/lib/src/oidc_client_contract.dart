import 'models.dart';

class OidcAuthorizationResult {
  const OidcAuthorizationResult({required this.accessToken});

  final String accessToken;
}

abstract class OidcAuthorizationClient {
  Future<OidcAuthorizationResult> authorize(RuntimeConfig config);
}
