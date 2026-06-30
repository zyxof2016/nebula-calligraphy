import 'package:flutter/material.dart';

import 'src/app.dart';

void main() {
  const apiBaseUrl = String.fromEnvironment(
    'CALLIGRAPHY_API_BASE_URL',
    defaultValue: 'http://localhost:8090',
  );

  runApp(const CalligraphyApp(apiBaseUrl: apiBaseUrl));
}
