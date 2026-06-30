import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:nebula_calligraphy_app/src/calligraphy_api.dart';
import 'package:nebula_calligraphy_app/src/models.dart';

void main() {
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
    expect(result.glyphSizeCm, 25);
  });
}
