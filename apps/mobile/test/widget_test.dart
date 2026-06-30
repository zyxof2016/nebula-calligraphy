import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:nebula_calligraphy_app/src/app.dart';
import 'package:nebula_calligraphy_app/src/app_controller.dart';

import 'app_controller_test.dart';

void main() {
  testWidgets('renders unauthenticated calligraphy login screen', (
    WidgetTester tester,
  ) async {
    final controller = CalligraphyController(
      gateway: FakeCalligraphyGateway(),
      apiBaseUrl: 'http://calligraphy.test',
    );

    await tester.pumpWidget(CalligraphyApp(controller: controller));

    expect(find.text('星云书法'), findsOneWidget);
    expect(find.text('登录'), findsOneWidget);
    expect(find.text('注册'), findsOneWidget);
  });

  testWidgets('renders daily learning workspace after login', (
    WidgetTester tester,
  ) async {
    final controller = CalligraphyController(
      gateway: FakeCalligraphyGateway(),
      apiBaseUrl: 'http://calligraphy.test',
    );
    await controller.login(username: 'learner', password: 'password123');

    await tester.pumpWidget(CalligraphyApp(controller: controller));
    await tester.pump();

    expect(find.text('今日'), findsWidgets);
    expect(find.text('查字'), findsWidgets);
    expect(find.text('创作'), findsWidgets);
  });

  testWidgets('loads a default practice reference after login', (
    WidgetTester tester,
  ) async {
    final controller = CalligraphyController(
      gateway: FakeCalligraphyGateway(),
      apiBaseUrl: 'http://calligraphy.test',
    );

    await controller.login(username: 'learner', password: 'password123');
    await tester.pumpWidget(CalligraphyApp(controller: controller));
    await tester.pump();

    expect(find.text('今日临摹：永 · 欧体'), findsOneWidget);
    expect(find.text('看帖 → 拆笔画 → 练结构 → 创章法'), findsOneWidget);
  });

  testWidgets('daily page exposes beginner-friendly primary actions', (
    WidgetTester tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(390, 844));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final controller = CalligraphyController(
      gateway: FakeCalligraphyGateway(),
      apiBaseUrl: 'http://calligraphy.test',
    );
    await controller.login(username: 'learner', password: 'password123');

    await tester.pumpWidget(CalligraphyApp(controller: controller));
    await tester.pump();

    expect(find.text('永 · 欧体'), findsOneWidget);
    expect(find.text('米字格'), findsOneWidget);
    expect(find.text('我已临摹'), findsWidgets);
    expect(find.text('换字'), findsOneWidget);
    expect(find.text('查字'), findsWidgets);
    expect(find.text('创作'), findsWidgets);
  });

  testWidgets('recording practice gives feedback and next step', (
    WidgetTester tester,
  ) async {
    final controller = CalligraphyController(
      gateway: FakeCalligraphyGateway(),
      apiBaseUrl: 'http://calligraphy.test',
    );
    await controller.login(username: 'learner', password: 'password123');

    await tester.pumpWidget(CalligraphyApp(controller: controller));
    await tester.pump();
    await tester.tap(find.widgetWithText(FilledButton, '我已临摹'));
    await tester.pump();

    expect(find.textContaining('已记录 ✓'), findsOneWidget);
    expect(find.textContaining('继续练“水”'), findsOneWidget);
    expect(find.text('今日已练 1 次'), findsOneWidget);
  });

  testWidgets('shows a copybook reference glyph for practice', (
    WidgetTester tester,
  ) async {
    final controller = CalligraphyController(
      gateway: FakeCalligraphyGateway(),
      apiBaseUrl: 'http://calligraphy.test',
    );
    await controller.login(username: 'learner', password: 'password123');
    await controller.searchGlyphs('永');

    await tester.pumpWidget(CalligraphyApp(controller: controller));
    await tester.pump();

    expect(find.text('今日临摹：永 · 欧体'), findsOneWidget);
    expect(find.text('出自《九成宫》'), findsOneWidget);
    expect(find.text('jiuchenggong'), findsNothing);
    expect(find.text('regular_ou'), findsNothing);
    expect(find.text('米字格'), findsOneWidget);
    expect(find.text('九宫格'), findsOneWidget);
    expect(find.text('双钩'), findsOneWidget);
  });

  testWidgets('daily workspace emphasizes calligraphy learning pillars', (
    WidgetTester tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(390, 844));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final controller = CalligraphyController(
      gateway: FakeCalligraphyGateway(),
      apiBaseUrl: 'http://calligraphy.test',
    );
    await controller.login(username: 'learner', password: 'password123');
    await controller.searchGlyphs('永');

    await tester.pumpWidget(CalligraphyApp(controller: controller));
    await tester.pump();

    expect(find.text('永 · 欧体'), findsOneWidget);
    expect(find.text('展开笔画、结构和章法'), findsOneWidget);
    await tester.tap(find.text('展开笔画、结构和章法'));
    await tester.pumpAndSettle();
    expect(find.text('基本笔画'), findsOneWidget);
    expect(find.text('单字结构'), findsOneWidget);
    expect(find.text('多字章法'), findsOneWidget);
    expect(find.text('点'), findsOneWidget);
    expect(find.text('横折钩'), findsOneWidget);
    expect(find.text('横撇'), findsOneWidget);
    expect(find.text('竖'), findsNothing);
    expect(find.text('中宫'), findsOneWidget);
    expect(find.text('结构辅助线'), findsOneWidget);
    expect(find.text('章法缩略图'), findsOneWidget);
    expect(find.text('查名家写法'), findsWidgets);
    expect(find.text('生成作品布局'), findsWidgets);
  });

  testWidgets('creation page starts with a paper-shaped empty preview', (
    WidgetTester tester,
  ) async {
    final controller = CalligraphyController(
      gateway: FakeCalligraphyGateway(),
      apiBaseUrl: 'http://calligraphy.test',
    );
    await controller.login(username: 'learner', password: 'password123');

    await tester.pumpWidget(CalligraphyApp(controller: controller));
    await tester.pump();
    await tester.tap(find.text('创作'));
    await tester.pump();

    expect(find.text('创作'), findsWidgets);
    expect(find.text('生成作品布局'), findsOneWidget);
    expect(find.text('输入内容后点击生成作品布局'), findsOneWidget);
  });
}
