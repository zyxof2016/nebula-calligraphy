import 'package:flutter_test/flutter_test.dart';
import 'package:nebula_calligraphy_app/src/app_controller.dart';
import 'package:nebula_calligraphy_app/src/calligraphy_api.dart';
import 'package:nebula_calligraphy_app/src/models.dart';
import 'package:nebula_calligraphy_app/src/session_store.dart';

void main() {
  test('login, search and preview update the learning workspace', () async {
    final gateway = FakeCalligraphyGateway();
    final controller = CalligraphyController(
      gateway: gateway,
      sessionStore: MemorySessionStore(),
      apiBaseUrl: 'http://calligraphy.test',
    );

    await controller.login(username: 'learner', password: 'password123');
    await controller.searchGlyphs('永');
    await controller.previewCreation(text: '山高月小');

    expect(controller.currentUser?.username, 'learner');
    expect(controller.glyphs.single.character, '永');
    expect(controller.layoutPreview?.normalizedText, '山高月小');
    expect(gateway.bearerToken, 'session-token');
    expect(gateway.lastSearchStyle, 'ou');
    expect(gateway.lastLayoutStyle, 'ou');
  });
}

class FakeCalligraphyGateway implements CalligraphyGateway {
  String? bearerToken;
  String? lastSearchStyle;
  String? lastLayoutStyle;
  int practiceCount = 0;

  @override
  void setBearerToken(String? token) {
    bearerToken = token;
  }

  @override
  Future<AuthSession> login({
    required String username,
    required String password,
  }) async {
    return AuthSession(
      token: 'session-token',
      user: User(userId: 'user-1', username: username, createdAt: 'now'),
    );
  }

  @override
  Future<AuthSession> register({
    required String username,
    required String password,
  }) async {
    return login(username: username, password: password);
  }

  @override
  Future<List<GlyphSummary>> searchGlyphs({
    String? character,
    String? style,
    String? copybookId,
  }) async {
    lastSearchStyle = style;
    return [
      GlyphSummary(
        glyphId: 'ou-yong-001',
        character: character ?? '永',
        style: style ?? 'regular_ou',
        copybookId: copybookId ?? 'jiuchenggong',
        calligrapher: '欧阳询',
        sourceImage: '',
        licenseStatus: 'public_domain',
        reviewStatus: 'published',
      ),
    ];
  }

  @override
  Future<List<GlyphPresetGroup>> listGlyphPresets({String? style}) async {
    return const [];
  }

  @override
  Future<GlyphDetail> getGlyphDetail(String glyphId) async {
    return GlyphDetail(
      glyph: GlyphSummary(
        glyphId: glyphId,
        character: '永',
        style: 'regular_ou',
        copybookId: 'jiuchenggong',
        calligrapher: '欧阳询',
        sourceImage: '',
        licenseStatus: 'public_domain',
        reviewStatus: 'published',
      ),
      structureNotes: const ['横竖居中，撇捺舒展。'],
      brushworkNotes: const ['藏锋起笔，中锋行笔。'],
      practiceTemplates: const [],
    );
  }

  @override
  Future<LayoutResult> previewLayout(LayoutRequest request) async {
    lastLayoutStyle = request.style;
    final normalizedText = request.text.replaceAll(RegExp(r'\s+'), '');
    final characters = normalizedText.split('');
    final rows = characters.length <= 4 ? 2 : 4;
    final columns = (characters.length / rows).ceil();
    final cellWidth =
        (request.paper.widthCm - request.marginCm * 2 - 6) / columns;
    final cellHeight = (request.paper.heightCm - request.marginCm * 2) / rows;
    final glyphSize = cellWidth < cellHeight
        ? cellWidth * 0.7
        : cellHeight * 0.68;
    final slots = <GlyphSlot>[];
    for (var i = 0; i < characters.length; i += 1) {
      final column = i ~/ rows;
      final row = i % rows;
      slots.add(
        GlyphSlot(
          index: i,
          character: characters[i],
          column: column,
          row: row,
          xCm:
              request.paper.widthCm -
              request.marginCm -
              6 -
              cellWidth * (column + 0.5),
          yCm: request.marginCm + cellHeight * (row + 0.5),
          sizeCm: glyphSize,
        ),
      );
    }
    return LayoutResult(
      layoutId: 'layout-1',
      normalizedText: normalizedText,
      characterCount: characters.length,
      style: request.style,
      copybookId: request.copybookId,
      paper: request.paper,
      direction: request.direction,
      marginCm: request.marginCm,
      columns: columns,
      rows: rows,
      glyphSizeCm: glyphSize,
      slots: slots,
      signatureSlots: const [
        TextSlot(index: 0, text: '六', xCm: 5.5, yCm: 7, sizeCm: 1.2),
        TextSlot(index: 1, text: '月', xCm: 5.5, yCm: 9, sizeCm: 1.2),
        TextSlot(index: 2, text: '试', xCm: 5.5, yCm: 11, sizeCm: 1.2),
        TextSlot(index: 3, text: '书', xCm: 5.5, yCm: 13, sizeCm: 1.2),
      ],
      sealSlots: const [
        TextSlot(index: 0, text: 'seal', xCm: 5.5, yCm: 62, sizeCm: 1.8),
      ],
    );
  }

  @override
  Future<ArtworkDraft> createDraft(CreateArtworkDraftRequest request) async {
    throw UnimplementedError();
  }

  @override
  Future<List<ArtworkDraft>> listDrafts() async {
    return const [];
  }

  @override
  Future<ExportRecord> exportDraft({
    required String artworkId,
    required String format,
    required String templateType,
  }) async {
    throw UnimplementedError();
  }

  @override
  Future<LearningProfile> getLearningProfile(String ownerUserId) async {
    return LearningProfile(
      ownerUserId: ownerUserId,
      favorites: const [],
      recentPractice: const [],
      practiceCount: practiceCount,
      favoriteCount: 0,
    );
  }

  @override
  Future<PracticeRecord> recordPractice({
    required String ownerUserId,
    required String glyphId,
    required String templateType,
    required String gridType,
  }) async {
    practiceCount += 1;
    return PracticeRecord(
      practiceId: 'practice-$practiceCount',
      ownerUserId: ownerUserId,
      glyphId: glyphId,
      character: '永',
      style: 'ou',
      templateType: templateType,
      gridType: gridType,
      createdAt: 'now',
    );
  }
}
