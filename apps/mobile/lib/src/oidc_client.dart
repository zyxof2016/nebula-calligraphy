import 'oidc_client_contract.dart';
import 'oidc_client_stub.dart'
    if (dart.library.io) 'oidc_client_native.dart'
    as implementation;

export 'oidc_client_contract.dart';

OidcAuthorizationClient createOidcAuthorizationClient() =>
    implementation.createOidcAuthorizationClient();
