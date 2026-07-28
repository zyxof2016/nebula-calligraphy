import 'dart:io';

import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:nebula_calligraphy_app/src/app.dart';
import 'package:nebula_calligraphy_app/src/app_controller.dart';

import 'app_controller_test.dart';

void main() {
  setUpAll(() async {
    for (final candidate in [
      '/usr/share/fonts/truetype/wqy/wqy-microhei.ttc',
      '/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc',
      '/tmp/calligraphy-test-fonts/extracted/usr/share/fonts/truetype/wqy/wqy-microhei.ttc',
    ]) {
      final file = File(candidate);
      if (await file.exists()) {
        final uiFont = await file.readAsBytes();
        final uiLoader = FontLoader('WenQuanYi Micro Hei')
          ..addFont(Future.value(ByteData.sublistView(uiFont)));
        await uiLoader.load();
        break;
      }
    }
    final calligraphyFont = await File(
      '../../assets/fonts/MaShanZheng-Regular.ttf',
    ).readAsBytes();
    final calligraphyLoader = FontLoader('KaiTi')
      ..addFont(Future.value(ByteData.sublistView(calligraphyFont)));
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
