import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:nebula_calligraphy_app/src/app.dart';
import 'package:nebula_calligraphy_app/src/app_controller.dart';

import 'app_controller_test.dart';

void main() {
  setUpAll(() async {
    final calligraphyLoader = FontLoader('MaShanZheng')
      ..addFont(rootBundle.load('assets/fonts/MaShanZheng-Regular.ttf'));
    await calligraphyLoader.load();
    final materialIcons = FontLoader('MaterialIcons')
      ..addFont(rootBundle.load('fonts/MaterialIcons-Regular.otf'));
    await materialIcons.load();
  });

  testWidgets('capture daily practice mobile visual', (tester) async {
    await tester.binding.setSurfaceSize(const Size(390, 844));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    final controller = CalligraphyController(
      gateway: FakeCalligraphyGateway(),
      apiBaseUrl: 'http://calligraphy.test',
    );
    await controller.login(username: 'learner', password: 'password123');
    await controller.searchGlyphs('永');

    await tester.pumpWidget(CalligraphyApp(controller: controller));
    await tester.pumpAndSettle();

    await expectLater(
      find.byType(CalligraphyApp),
      matchesGoldenFile('goldens/daily_practice_mobile.png'),
    );
  });

  testWidgets('capture daily practice desktop visual', (tester) async {
    await tester.binding.setSurfaceSize(const Size(1280, 900));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    final controller = CalligraphyController(
      gateway: FakeCalligraphyGateway(),
      apiBaseUrl: 'http://calligraphy.test',
    );
    await controller.login(username: 'learner', password: 'password123');
    await controller.searchGlyphs('永');

    await tester.pumpWidget(CalligraphyApp(controller: controller));
    await tester.pumpAndSettle();

    await expectLater(
      find.byType(CalligraphyApp),
      matchesGoldenFile('goldens/daily_practice_desktop.png'),
    );
  });

  testWidgets('capture creation layout mobile visual', (tester) async {
    await tester.binding.setSurfaceSize(const Size(390, 844));
    addTearDown(() => tester.binding.setSurfaceSize(null));

    final controller = CalligraphyController(
      gateway: FakeCalligraphyGateway(),
      apiBaseUrl: 'http://calligraphy.test',
    );
    await controller.login(username: 'learner', password: 'password123');
    await controller.previewCreation(text: '山高月小 水落石出');

    await tester.pumpWidget(CalligraphyApp(controller: controller));
    await tester.pumpAndSettle();
    await tester.tap(find.text('创作').last);
    await tester.pumpAndSettle();

    await expectLater(
      find.byType(CalligraphyApp),
      matchesGoldenFile('goldens/creation_layout_mobile.png'),
    );
  });
}
