import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

import 'src/app.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  const configuredApiBaseUrl = String.fromEnvironment(
    'CALLIGRAPHY_API_BASE_URL',
  );
  final apiBaseUrl = configuredApiBaseUrl.isNotEmpty
      ? configuredApiBaseUrl
      : kIsWeb
      ? Uri.base.origin
      : 'http://localhost:8090';
  final apiUri = Uri.tryParse(apiBaseUrl);
  if (kReleaseMode &&
      (apiUri == null || apiUri.scheme != 'https' || !apiUri.hasAuthority)) {
    throw StateError(
      'Release builds require an HTTPS CALLIGRAPHY_API_BASE_URL.',
    );
  }

  runApp(CalligraphyApp(apiBaseUrl: apiBaseUrl));
}
