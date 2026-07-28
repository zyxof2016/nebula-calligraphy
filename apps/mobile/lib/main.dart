import 'package:flutter/material.dart';
import 'package:flutter/foundation.dart';

import 'src/app.dart';

void main() {
  const configuredApiBaseUrl = String.fromEnvironment(
    'CALLIGRAPHY_API_BASE_URL',
  );
  final apiBaseUrl = configuredApiBaseUrl.isNotEmpty
      ? configuredApiBaseUrl
      : kIsWeb
      ? Uri.base.origin
      : 'http://localhost:8090';

  runApp(CalligraphyApp(apiBaseUrl: apiBaseUrl));
}
