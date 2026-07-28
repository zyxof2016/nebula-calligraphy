import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:nebula_calligraphy_app/src/calligraphy_api.dart';
import 'package:nebula_calligraphy_app/src/models.dart';

void main() {
  test('login decodes expiring auth session', () async {
    final api = CalligraphyApi(
      baseUrl: Uri.parse('http://calligraphy.test'),
      client: MockClient((request) async {
        if (request.url.path == '/api/v1/calligraphy/runtime-config') {
          return http.Response(
            jsonEncode({'runtime_profile': 'trial', 'auth_mode': 'local'}),
            200,
          );
        }
        expect(request.method, 'POST');
        expect(request.url.path, '/api/v1/calligraphy/auth/login');
        return http.Response(
          jsonEncode({
            'token': 'session-token',
            'expires_at': '2026-07-29T10:00:00Z',
            'user': {
              'user_id': 'user-1',
              'username': 'learner',
              'created_at': '2026-07-28T10:00:00Z',
            },
          }),
          200,
          headers: {'content-type': 'application/json'},
        );
      }),
    );

    final session = await api.login(username: 'learner', password: 'secret123');

    expect(session.token, 'session-token');
    expect(session.expiresAt, '2026-07-29T10:00:00Z');
  });

  test('managed login uses Nebula Identity direct endpoint', () async {
    final jwt = [
      base64Url
          .encode(utf8.encode(jsonEncode({'alg': 'RS256'})))
          .replaceAll('=', ''),
      base64Url
          .encode(
            utf8.encode(
              jsonEncode({
                'sub': 'user-1',
                'exp':
                    DateTime.now()
                        .toUtc()
                        .add(const Duration(hours: 1))
                        .millisecondsSinceEpoch ~/
                    1000,
              }),
            ),
          )
          .replaceAll('=', ''),
      'signature',
    ].join('.');
    final api = CalligraphyApi(
      baseUrl: Uri.parse('https://calligraphy.test'),
      client: MockClient((request) async {
        if (request.url.path == '/api/v1/calligraphy/runtime-config') {
          return http.Response(
            jsonEncode({
              'runtime_profile': 'managed',
              'auth_mode': 'nebula-direct',
              'identity_login_endpoint':
                  'https://identity.test/api/v1/auth/login',
            }),
            200,
          );
        }
        if (request.url.host == 'identity.test') {
          expect(request.method, 'POST');
          return http.Response(jsonEncode({'access_token': jwt}), 200);
        }
        if (request.url.path == '/api/v1/calligraphy/auth/me') {
          expect(request.headers['authorization'], 'Bearer $jwt');
          return http.Response(
            jsonEncode({
              'user_id': 'user-1',
              'username': 'learner',
              'created_at': '',
            }),
            200,
          );
        }
        return http.Response('not found', 404);
      }),
    );

    final session = await api.login(username: 'learner', password: 'secret123');

    expect(session.token, jwt);
    expect(session.user.userId, 'user-1');
    expect(DateTime.parse(session.expiresAt).isAfter(DateTime.now()), isTrue);
  });

  test('identity non-json failure remains a readable API error', () async {
    final client = MockClient((request) async {
      if (request.url.path == '/api/v1/calligraphy/runtime-config') {
        return http.Response(
          jsonEncode({
            'runtime_profile': 'managed',
            'auth_mode': 'nebula-direct',
            'identity_login_endpoint': 'https://identity.example/login',
          }),
          200,
        );
      }
      return http.Response('upstream unavailable', 503);
    });
    final api = CalligraphyApi(
      baseUrl: Uri.parse('https://calligraphy.example'),
      client: client,
    );

    await expectLater(
      api.login(username: 'learner', password: 'secret123'),
      throwsA(
        isA<ApiException>()
            .having((error) => error.statusCode, 'statusCode', 503)
            .having(
              (error) => error.message,
              'message',
              'upstream unavailable',
            ),
      ),
    );
  });

  test('searchGlyphs decodes glyph list from the service contract', () async {
    final api = CalligraphyApi(
      baseUrl: Uri.parse('http://calligraphy.test'),
      client: MockClient((request) async {
        expect(request.method, 'GET');
        expect(request.url.path, '/api/v1/calligraphy/glyphs/search');
        expect(request.url.queryParameters['character'], '永');
        return http.Response(
          jsonEncode({
            'items': [
              {
                'glyph_id': 'ou-yong-001',
                'character': '永',
                'style': 'regular_ou',
                'copybook_id': 'jiuchenggong',
                'calligrapher': '欧阳询',
                'source_image': 'copybooks/jiuchenggong/yong.png',
                'render_asset': {
                  'url':
                      'http://calligraphy.test/api/v1/calligraphy/glyphs/ou-yong-001/render.png',
                  'content_type': 'image/png',
                  'width': 512,
                  'height': 512,
                  'source': 'server_font',
                },
                'crop_box': {
                  'x': 0,
                  'y': 0,
                  'width': 120,
                  'height': 120,
                  'unit': 'px',
                },
                'license_status': 'public_domain',
                'review_status': 'published',
              },
            ],
          }),
          200,
          headers: {'content-type': 'application/json'},
        );
      }),
    );

    final glyphs = await api.searchGlyphs(character: '永');

    expect(glyphs, hasLength(1));
    expect(glyphs.single.character, '永');
    expect(glyphs.single.style, 'regular_ou');
    expect(glyphs.single.copybookId, 'jiuchenggong');
    expect(glyphs.single.renderAsset.contentType, 'image/png');
    expect(glyphs.single.renderAsset.url, contains('/render.png'));
  });

  test('previewLayout posts layout request and decodes slot matrix', () async {
    final api = CalligraphyApi(
      baseUrl: Uri.parse('http://calligraphy.test'),
      client: MockClient((request) async {
        expect(request.method, 'POST');
        expect(request.url.path, '/api/v1/calligraphy/layouts/preview');
        final payload = jsonDecode(request.body) as Map<String, dynamic>;
        expect(payload['text'], '山高月小');
        expect(payload['direction'], 'vertical_rtl');
        expect(payload['paper']['format'], '斗方');
        return http.Response(
          jsonEncode({
            'layout_id': 'layout-1',
            'normalized_text': '山高月小',
            'character_count': 4,
            'style': 'regular_yan',
            'copybook_id': 'duobaota',
            'paper': {'format': '斗方', 'width_cm': 69, 'height_cm': 68},
            'direction': 'vertical_rtl',
            'margin_cm': 3,
            'columns': 2,
            'rows': 2,
            'glyph_size_cm': 25,
            'slots': [
              {
                'index': 0,
                'character': '山',
                'column': 0,
                'row': 0,
                'x_cm': 40,
                'y_cm': 8,
                'size_cm': 25,
                'render_asset': {
                  'url':
                      'http://calligraphy.test/api/v1/calligraphy/glyphs/ou-shan-001/render.png',
                  'content_type': 'image/png',
                  'width': 512,
                  'height': 512,
                  'source': 'server_font',
                },
              },
            ],
          }),
          200,
          headers: {'content-type': 'application/json'},
        );
      }),
    );

    final result = await api.previewLayout(
      LayoutRequest(
        text: '山高月小',
        style: 'regular_yan',
        copybookId: 'duobaota',
        paper: const PaperSpec(format: '斗方', widthCm: 69, heightCm: 68),
        signature: '六月 试书',
      ),
    );

    expect(result.layoutId, 'layout-1');
    expect(result.slots.single.character, '山');
    expect(result.slots.single.renderAsset.url, contains('/render.png'));
    expect(result.glyphSizeCm, 25);
  });

  test('exportDraft decodes png export metadata', () async {
    final api = CalligraphyApi(
      baseUrl: Uri.parse('http://calligraphy.test'),
      client: MockClient((request) async {
        expect(request.method, 'POST');
        expect(
          request.url.path,
          '/api/v1/calligraphy/artworks/drafts/artwork-1/exports',
        );
        final payload = jsonDecode(request.body) as Map<String, dynamic>;
        expect(payload['format'], 'png');
        return http.Response(
          jsonEncode({
            'export_id': 'export-1',
            'artwork_id': 'artwork-1',
            'format': 'png',
            'template_type': 'reference',
            'content_type': 'image/png',
            'sha256': 'abc123',
            'byte_size': 12,
            'inline_content': 'iVBORw0KGgo=',
            'inline_encoding': 'base64',
            'created_at': '2026-07-06T00:00:00Z',
          }),
          201,
          headers: {'content-type': 'application/json'},
        );
      }),
    );

    final export = await api.exportDraft(
      artworkId: 'artwork-1',
      format: 'png',
      templateType: 'reference',
    );

    expect(export.contentType, 'image/png');
    expect(export.inlineEncoding, 'base64');
    expect(export.inlineContent, startsWith('iVBOR'));
  });

  test('getLearningProfile decodes daily practice plan', () async {
    final api = CalligraphyApi(
      baseUrl: Uri.parse('http://calligraphy.test'),
      client: MockClient((request) async {
        expect(request.method, 'GET');
        expect(request.url.path, '/api/v1/calligraphy/users/user-1/learning');
        return http.Response(
          jsonEncode({
            'owner_user_id': 'user-1',
            'favorites': [],
            'recent_practice': [],
            'daily_plan': [
              {
                'glyph_id': 'ou-common-永',
                'character': '永',
                'style': 'ou',
                'copybook_id': 'common-practice-ou',
                'title': '基础笔画代表字',
                'reason': '巩固基础笔画和起收笔',
              },
            ],
            'daily_steps': [
              {
                'step_id': 'copy_reference',
                'title': '临摹今日字',
                'description': '先看参考字形，再写 3 遍。',
                'action_label': '开始临摹',
                'target_glyph_id': 'ou-common-永',
                'target_character': '永',
                'completed': false,
              },
            ],
            'practice_count': 0,
            'today_practice_count': 0,
            'favorite_count': 0,
          }),
          200,
          headers: {'content-type': 'application/json'},
        );
      }),
    );

    final profile = await api.getLearningProfile('user-1');

    expect(profile.dailyPlan, hasLength(1));
    expect(profile.dailyPlan.single.character, '永');
    expect(profile.dailyPlan.single.reason, contains('基础笔画'));
    expect(profile.dailySteps, hasLength(1));
    expect(profile.dailySteps.single.title, '临摹今日字');
    expect(profile.dailySteps.single.targetCharacter, '永');
    expect(profile.dailySteps.single.completed, isFalse);
    expect(profile.todayPracticeCount, 0);
  });
}
