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
    return LayoutResult(
      layoutId: 'layout-1',
      normalizedText: request.text,
      characterCount: request.text.length,
      style: request.style,
      copybookId: request.copybookId,
      paper: request.paper,
      direction: request.direction,
      marginCm: request.marginCm,
      columns: 2,
      rows: 2,
      glyphSizeCm: 25,
      slots: const [],
      signatureSlots: const [],
      sealSlots: const [],
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
