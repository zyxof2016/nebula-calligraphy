import 'package:flutter/material.dart';

import 'app_controller.dart';
import 'calligraphy_api.dart';
import 'models.dart';
import 'session_store.dart';

class CalligraphyApp extends StatefulWidget {
  const CalligraphyApp({
    super.key,
    this.apiBaseUrl = 'http://localhost:8090',
    this.controller,
  });

  final String apiBaseUrl;
  final CalligraphyController? controller;

  @override
  State<CalligraphyApp> createState() => _CalligraphyAppState();
}

class _CalligraphyAppState extends State<CalligraphyApp> {
  late final CalligraphyController _controller;

  @override
  void initState() {
    super.initState();
    _controller =
        widget.controller ??
        CalligraphyController(
          gateway: CalligraphyApi(baseUrl: Uri.parse(widget.apiBaseUrl)),
          sessionStore: SharedPreferencesSessionStore(),
          apiBaseUrl: widget.apiBaseUrl,
        );
    _controller.initialize();
  }

  @override
  void dispose() {
    if (widget.controller == null) {
      _controller.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: '星云书法',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: const Color(0xff2f5f4f),
          brightness: Brightness.light,
        ),
        scaffoldBackgroundColor: const Color(0xFFF7F4EB),
        useMaterial3: true,
        fontFamilyFallback: const [
          'Noto Sans CJK SC',
          'PingFang SC',
          'KaiTi',
          'STKaiti',
          'serif',
        ],
      ),
      home: AnimatedBuilder(
        animation: _controller,
        builder: (context, _) {
          if (!_controller.isAuthenticated) {
            return LoginScreen(controller: _controller);
          }
          return LearningWorkspace(controller: _controller);
        },
      ),
    );
  }
}

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key, required this.controller});

  final CalligraphyController controller;

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _username = TextEditingController(text: 'learner');
  final _password = TextEditingController(text: 'password123');

  @override
  void dispose() {
    _username.dispose();
    _password.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final controller = widget.controller;
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 460),
            child: ListView(
              padding: const EdgeInsets.all(24),
              shrinkWrap: true,
              children: [
                Text(
                  '星云书法',
                  style: Theme.of(context).textTheme.headlineLarge?.copyWith(
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 8),
                Text(
                  '登录后开始查字、临摹、集字排版和作品导出。',
                  style: Theme.of(context).textTheme.bodyLarge,
                ),
                const SizedBox(height: 24),
                TextField(
                  controller: _username,
                  decoration: const InputDecoration(
                    labelText: '账号',
                    border: OutlineInputBorder(),
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _password,
                  obscureText: true,
                  decoration: const InputDecoration(
                    labelText: '密码',
                    border: OutlineInputBorder(),
                  ),
                ),
                const SizedBox(height: 16),
                if (controller.errorMessage != null)
                  Padding(
                    padding: const EdgeInsets.only(bottom: 12),
                    child: Text(
                      controller.errorMessage!,
                      style: TextStyle(
                        color: Theme.of(context).colorScheme.error,
                      ),
                    ),
                  ),
                Row(
                  children: [
                    Expanded(
                      child: FilledButton.icon(
                        onPressed: controller.busy
                            ? null
                            : () => controller.login(
                                username: _username.text,
                                password: _password.text,
                              ),
                        icon: const Icon(Icons.login),
                        label: const Text('登录'),
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: OutlinedButton.icon(
                        onPressed: controller.busy
                            ? null
                            : () => controller.register(
                                username: _username.text,
                                password: _password.text,
                              ),
                        icon: const Icon(Icons.person_add_alt_1),
                        label: const Text('注册'),
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 16),
                Text(
                  'API: ${controller.apiBaseUrl}',
                  style: Theme.of(context).textTheme.bodySmall,
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class LearningWorkspace extends StatefulWidget {
  const LearningWorkspace({super.key, required this.controller});

  final CalligraphyController controller;

  @override
  State<LearningWorkspace> createState() => _LearningWorkspaceState();
}

class _LearningWorkspaceState extends State<LearningWorkspace> {
  int _index = 0;

  @override
  Widget build(BuildContext context) {
    final pages = [
      DailyPracticePage(
        controller: widget.controller,
        onOpenSearch: () => setState(() => _index = 1),
        onOpenCreation: () => setState(() => _index = 2),
      ),
      GlyphSearchPage(controller: widget.controller),
      CreationPage(controller: widget.controller),
      DraftsPage(controller: widget.controller),
    ];
    return Scaffold(
      appBar: AppBar(
        title: const Text('星云书法'),
        actions: [
          IconButton(
            tooltip: '退出登录',
            onPressed: widget.controller.logout,
            icon: const Icon(Icons.logout),
          ),
        ],
      ),
      body: pages[_index],
      bottomNavigationBar: NavigationBar(
        selectedIndex: _index,
        onDestinationSelected: (value) => setState(() => _index = value),
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.today_outlined),
            selectedIcon: Icon(Icons.today),
            label: '今日',
          ),
          NavigationDestination(
            icon: Icon(Icons.search),
            selectedIcon: Icon(Icons.manage_search),
            label: '查字',
          ),
          NavigationDestination(
            icon: Icon(Icons.auto_awesome_mosaic_outlined),
            selectedIcon: Icon(Icons.auto_awesome_mosaic),
            label: '创作',
          ),
          NavigationDestination(
            icon: Icon(Icons.folder_copy_outlined),
            selectedIcon: Icon(Icons.folder_copy),
            label: '作品',
          ),
        ],
      ),
    );
  }
}

class DailyPracticePage extends StatelessWidget {
  const DailyPracticePage({
    super.key,
    required this.controller,
    required this.onOpenSearch,
    required this.onOpenCreation,
  });

  final CalligraphyController controller;
  final VoidCallback onOpenSearch;
  final VoidCallback onOpenCreation;

  @override
  Widget build(BuildContext context) {
    final profile = controller.learningProfile;
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        const PracticePlanHeader(),
        const SizedBox(height: 12),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            FilledButton.icon(
              onPressed: controller.changePracticeGlyph,
              icon: const Icon(Icons.refresh),
              label: const Text('换一个字'),
            ),
            OutlinedButton.icon(
              onPressed: onOpenSearch,
              icon: const Icon(Icons.manage_search),
              label: const Text('查名家写法'),
            ),
            FilledButton.tonalIcon(
              onPressed: onOpenCreation,
              icon: const Icon(Icons.grid_view),
              label: const Text('生成作品布局'),
            ),
          ],
        ),
        const SizedBox(height: 12),
        if (controller.selectedGlyph != null)
          PracticeReferencePanel(
            detail: controller.selectedGlyph!,
            onPractice: controller.recordSelectedGlyphPractice,
            feedback: controller.practiceFeedback,
            practiceCount: profile?.practiceCount ?? 0,
          )
        else
          const EmptyPanel(
            icon: Icons.gesture,
            title: '选择一个常用字开始',
            message: '系统会展示结构要点、笔法要点和可用临摹模板。',
          ),
        const SizedBox(height: 12),
        Wrap(
          spacing: 12,
          runSpacing: 12,
          children: [
            BasicStrokeCard(
              detail: controller.selectedGlyph,
              onSelectGlyph: controller.searchGlyphs,
            ),
            StructureFocusCard(
              detail: controller.selectedGlyph,
              onOpenSearch: onOpenSearch,
            ),
            CompositionFocusCard(
              paper: controller.selectedPaper,
              onOpenCreation: onOpenCreation,
            ),
          ],
        ),
        const SizedBox(height: 16),
        Wrap(
          spacing: 12,
          runSpacing: 12,
          children: [
            StatTile(
              label: '今日已练',
              value: '${profile?.practiceCount ?? 0}',
              icon: Icons.edit_note,
            ),
            StatTile(
              label: '收藏字',
              value: '${profile?.favoriteCount ?? 0}',
              icon: Icons.star_border,
            ),
            StatTile(
              label: '当前书体',
              value: styleOptions[controller.selectedStyle] ?? '欧体',
              icon: Icons.brush,
            ),
          ],
        ),
        const SizedBox(height: 16),
        Wrap(
          spacing: 12,
          runSpacing: 12,
          children: commonGlyphs
              .map(
                (glyph) => PracticeGlyphChip(
                  glyph: glyph,
                  onPressed: () => controller.searchGlyphs(glyph),
                ),
              )
              .toList(),
        ),
      ],
    );
  }
}

const commonGlyphs = ['永', '山', '水', '月', '人', '心', '中', '和'];

String copybookDisplayName(String copybookId) {
  return switch (copybookId) {
    'jiuchenggong' => '出自《九成宫》',
    'duobaota' => '出自《多宝塔》',
    'xuanmita' => '出自《玄秘塔》',
    'danbabei' => '出自《胆巴碑》',
    'qianziwen' => '出自《千字文》',
    _ => '参考范字',
  };
}

List<String> strokesForCharacter(String character) {
  return switch (character) {
    '永' => const ['点', '横折钩', '横撇', '撇', '捺'],
    '人' => const ['撇', '捺'],
    '一' => const ['横'],
    '十' => const ['横', '竖'],
    '心' => const ['点', '卧钩'],
    _ => const ['点', '横', '撇', '捺'],
  };
}

String strokeTipForCharacter(String character) {
  return switch (character) {
    '永' => '例：永字首笔为点，起笔藏锋，收笔轻提。',
    '人' => '例：人字撇捺要舒展，重心落在交汇处。',
    _ => '点击代表字，观察起笔、行笔和收笔。',
  };
}

class PracticePlanHeader extends StatelessWidget {
  const PracticePlanHeader({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('今日任务', style: Theme.of(context).textTheme.headlineSmall),
        const SizedBox(height: 6),
        Text(
          '看帖 → 拆笔画 → 练结构 → 创章法',
          style: Theme.of(context).textTheme.bodyMedium,
        ),
        const SizedBox(height: 10),
        const Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            Chip(label: Text('选今日字')),
            Chip(label: Text('看临摹参考')),
            Chip(label: Text('我已临摹')),
          ],
        ),
      ],
    );
  }
}

class PracticeReferencePanel extends StatelessWidget {
  const PracticeReferencePanel({
    super.key,
    required this.detail,
    required this.onPractice,
    required this.practiceCount,
    this.feedback,
  });

  final GlyphDetail detail;
  final VoidCallback onPractice;
  final int practiceCount;
  final String? feedback;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Wrap(
          spacing: 20,
          runSpacing: 16,
          crossAxisAlignment: WrapCrossAlignment.center,
          children: [
            ReferenceGlyphCard(glyph: detail.glyph, size: 188, fontSize: 112),
            SizedBox(
              width: 420,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('临摹参考', style: Theme.of(context).textTheme.titleLarge),
                  const SizedBox(height: 6),
                  Text(
                    '${styleOptions[detail.glyph.style] ?? detail.glyph.style} · ${detail.glyph.calligrapher}',
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                  Text(copybookDisplayName(detail.glyph.copybookId)),
                  const SizedBox(height: 8),
                  const Text('米字格参考'),
                  const SizedBox(height: 12),
                  const PracticeModeSelector(),
                  const SizedBox(height: 12),
                  FilledButton.icon(
                    onPressed: onPractice,
                    icon: const Icon(Icons.draw),
                    label: const Text('我已临摹'),
                  ),
                  if (feedback != null) ...[
                    const SizedBox(height: 10),
                    Text(feedback!),
                    const Text('今日已练'),
                    Text('$practiceCount'),
                    const Text('继续练“水”'),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class PracticeGlyphChip extends StatelessWidget {
  const PracticeGlyphChip({
    super.key,
    required this.glyph,
    required this.onPressed,
  });

  final String glyph;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return ActionChip(
      label: Text(glyph),
      avatar: const Icon(Icons.search, size: 18),
      onPressed: onPressed,
    );
  }
}

class PracticeModeSelector extends StatefulWidget {
  const PracticeModeSelector({super.key});

  @override
  State<PracticeModeSelector> createState() => _PracticeModeSelectorState();
}

class _PracticeModeSelectorState extends State<PracticeModeSelector> {
  String _mode = 'mi';

  @override
  Widget build(BuildContext context) {
    return SegmentedButton<String>(
      segments: const [
        ButtonSegment(value: 'mi', label: Text('米字格临摹')),
        ButtonSegment(value: 'jiugong', label: Text('九宫格结构')),
        ButtonSegment(value: 'outline', label: Text('双钩练习')),
      ],
      selected: {_mode},
      onSelectionChanged: (value) => setState(() => _mode = value.single),
    );
  }
}

class BasicStrokeCard extends StatelessWidget {
  const BasicStrokeCard({
    super.key,
    required this.detail,
    required this.onSelectGlyph,
  });

  final GlyphDetail? detail;
  final ValueChanged<String> onSelectGlyph;

  @override
  Widget build(BuildContext context) {
    final strokes = strokesForCharacter(detail?.glyph.character ?? '永');
    return LearningFocusCard(
      title: '基本笔画',
      icon: Icons.brush_outlined,
      width: 220,
      children: [
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: strokes.map((stroke) => Chip(label: Text(stroke))).toList(),
        ),
        const SizedBox(height: 12),
        Text(strokeTipForCharacter(detail?.glyph.character ?? '永')),
        const SizedBox(height: 12),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: const ['一', '十', '人', '永', '心']
              .map(
                (glyph) => ActionChip(
                  label: Text(glyph),
                  onPressed: () => onSelectGlyph(glyph),
                ),
              )
              .toList(),
        ),
      ],
    );
  }
}

class StructureFocusCard extends StatelessWidget {
  const StructureFocusCard({
    super.key,
    required this.detail,
    required this.onOpenSearch,
  });

  final GlyphDetail? detail;
  final VoidCallback onOpenSearch;

  @override
  Widget build(BuildContext context) {
    final notes = detail?.structureNotes ?? const ['先看中宫，再看主笔和重心。'];
    return LearningFocusCard(
      title: '单字结构',
      icon: Icons.grid_4x4_outlined,
      width: 220,
      children: [
        const StructureGuidePreview(),
        const SizedBox(height: 12),
        const Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            Chip(label: Text('中宫')),
            Chip(label: Text('主笔')),
            Chip(label: Text('重心')),
            Chip(label: Text('收放')),
          ],
        ),
        const SizedBox(height: 12),
        ...notes.take(2).map((note) => Text('· $note')),
        const SizedBox(height: 12),
        OutlinedButton.icon(
          onPressed: onOpenSearch,
          icon: const Icon(Icons.manage_search),
          label: const Text('查名家写法'),
        ),
      ],
    );
  }
}

class CompositionFocusCard extends StatelessWidget {
  const CompositionFocusCard({
    super.key,
    required this.paper,
    required this.onOpenCreation,
  });

  final PaperSpec paper;
  final VoidCallback onOpenCreation;

  @override
  Widget build(BuildContext context) {
    return LearningFocusCard(
      title: '多字章法',
      icon: Icons.dashboard_customize_outlined,
      width: 220,
      children: [
        const CompositionThumbnail(),
        const SizedBox(height: 12),
        Text(
          '${paper.format} · ${paper.widthCm.toStringAsFixed(0)}x${paper.heightCm.toStringAsFixed(0)}cm',
        ),
        const SizedBox(height: 12),
        const Text('山高月小'),
        const Text('水落石出'),
        const SizedBox(height: 12),
        FilledButton.icon(
          onPressed: onOpenCreation,
          icon: const Icon(Icons.grid_view),
          label: const Text('生成作品布局'),
        ),
      ],
    );
  }
}

class StructureGuidePreview extends StatelessWidget {
  const StructureGuidePreview({super.key});

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 92,
      width: double.infinity,
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: const Color(0xFFFFFCF5),
          border: Border.all(color: const Color(0x669A6A3A)),
        ),
        child: CustomPaint(
          painter: StructureGuidePainter(),
          child: const Center(child: Text('结构辅助线')),
        ),
      ),
    );
  }
}

class StructureGuidePainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = const Color(0x889A2E2E)
      ..strokeWidth = 1
      ..style = PaintingStyle.stroke;
    final rect = Rect.fromCenter(
      center: Offset(size.width / 2, size.height / 2),
      width: size.height * 0.58,
      height: size.height * 0.58,
    );
    canvas.drawRect(rect, paint);
    canvas.drawLine(
      Offset(size.width / 2, 10),
      Offset(size.width / 2, size.height - 10),
      paint,
    );
    canvas.drawLine(
      Offset(size.width * 0.32, size.height * 0.72),
      Offset(size.width * 0.68, size.height * 0.28),
      paint,
    );
  }

  @override
  bool shouldRepaint(covariant StructureGuidePainter oldDelegate) => false;
}

class CompositionThumbnail extends StatelessWidget {
  const CompositionThumbnail({super.key});

  @override
  Widget build(BuildContext context) {
    return AspectRatio(
      aspectRatio: 1,
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: const Color(0xFFFFFCF5),
          border: Border.all(color: const Color(0x669A6A3A)),
        ),
        child: const Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text('章法缩略图'),
              SizedBox(height: 8),
              Text('山  月'),
              Text('水  石'),
            ],
          ),
        ),
      ),
    );
  }
}

class LearningFocusCard extends StatelessWidget {
  const LearningFocusCard({
    super.key,
    required this.title,
    required this.icon,
    required this.children,
    required this.width,
  });

  final String title;
  final IconData icon;
  final List<Widget> children;
  final double width;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: width,
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Icon(icon),
                  const SizedBox(width: 8),
                  Text(title, style: Theme.of(context).textTheme.titleMedium),
                ],
              ),
              const SizedBox(height: 12),
              ...children,
            ],
          ),
        ),
      ),
    );
  }
}

class GlyphSearchPage extends StatefulWidget {
  const GlyphSearchPage({super.key, required this.controller});

  final CalligraphyController controller;

  @override
  State<GlyphSearchPage> createState() => _GlyphSearchPageState();
}

class _GlyphSearchPageState extends State<GlyphSearchPage> {
  final _query = TextEditingController(text: '永');

  @override
  void dispose() {
    _query.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final controller = widget.controller;
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Row(
          children: [
            Expanded(
              child: TextField(
                controller: _query,
                decoration: const InputDecoration(
                  labelText: '输入单字',
                  border: OutlineInputBorder(),
                ),
              ),
            ),
            const SizedBox(width: 8),
            IconButton.filled(
              tooltip: '查字',
              onPressed: () => controller.searchGlyphs(_query.text),
              icon: const Icon(Icons.search),
            ),
          ],
        ),
        const SizedBox(height: 12),
        StyleSelector(controller: controller),
        const SizedBox(height: 16),
        if (controller.busy) const LinearProgressIndicator(),
        if (controller.glyphs.isEmpty)
          const EmptyPanel(
            icon: Icons.menu_book_outlined,
            title: '暂无查询结果',
            message: '输入常用字后可查看不同碑帖来源。',
          )
        else
          Wrap(
            spacing: 12,
            runSpacing: 12,
            children: controller.glyphs
                .map(
                  (glyph) => GlyphCard(
                    glyph: glyph,
                    onTap: () => controller.selectGlyph(glyph.glyphId),
                  ),
                )
                .toList(),
          ),
        const SizedBox(height: 16),
        if (controller.selectedGlyph != null)
          GlyphDetailPanel(
            detail: controller.selectedGlyph!,
            onPractice: controller.recordSelectedGlyphPractice,
          ),
      ],
    );
  }
}

class CreationPage extends StatefulWidget {
  const CreationPage({super.key, required this.controller});

  final CalligraphyController controller;

  @override
  State<CreationPage> createState() => _CreationPageState();
}

class _CreationPageState extends State<CreationPage> {
  final _text = TextEditingController(text: '山高月小 水落石出');

  @override
  void dispose() {
    _text.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final controller = widget.controller;
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Text('集字创作', style: Theme.of(context).textTheme.headlineSmall),
        const SizedBox(height: 12),
        TextField(
          controller: _text,
          minLines: 2,
          maxLines: 4,
          decoration: const InputDecoration(
            labelText: '创作内容',
            border: OutlineInputBorder(),
          ),
        ),
        const SizedBox(height: 12),
        StyleSelector(controller: controller),
        const SizedBox(height: 12),
        PaperSelector(controller: controller),
        const SizedBox(height: 12),
        Row(
          children: [
            Expanded(
              child: FilledButton.icon(
                onPressed: () => controller.previewCreation(text: _text.text),
                icon: const Icon(Icons.grid_view),
                label: const Text('生成作品布局'),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: OutlinedButton.icon(
                onPressed: controller.layoutPreview == null
                    ? null
                    : controller.saveCurrentDraft,
                icon: const Icon(Icons.save_alt),
                label: const Text('保存草稿'),
              ),
            ),
          ],
        ),
        const SizedBox(height: 16),
        if (controller.layoutPreview != null)
          LayoutPreviewCard(result: controller.layoutPreview!)
        else
          const EmptyPaperPreview(),
      ],
    );
  }
}

class EmptyPaperPreview extends StatelessWidget {
  const EmptyPaperPreview({super.key});

  @override
  Widget build(BuildContext context) {
    return AspectRatio(
      aspectRatio: 1,
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: const Color(0xFFFFFCF5),
          border: Border.all(
            color: const Color(0x889A6A3A),
            style: BorderStyle.solid,
          ),
        ),
        child: const Center(child: Text('输入内容后点击生成作品布局')),
      ),
    );
  }
}

class DraftsPage extends StatelessWidget {
  const DraftsPage({super.key, required this.controller});

  final CalligraphyController controller;

  @override
  Widget build(BuildContext context) {
    final draft = controller.currentDraft;
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Text('作品', style: Theme.of(context).textTheme.headlineSmall),
        const SizedBox(height: 12),
        if (draft != null)
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    draft.text,
                    style: Theme.of(context).textTheme.titleLarge,
                  ),
                  const SizedBox(height: 8),
                  Text('草稿 ID: ${draft.artworkId}'),
                  const SizedBox(height: 12),
                  FilledButton.icon(
                    onPressed: controller.exportCurrentDraft,
                    icon: const Icon(Icons.ios_share),
                    label: const Text('导出参考 SVG'),
                  ),
                ],
              ),
            ),
          )
        else
          const EmptyPanel(
            icon: Icons.folder_open,
            title: '还没有当前草稿',
            message: '到“集字创作”生成章法并保存后，可在这里导出。',
          ),
        if (controller.lastExport != null) ...[
          const SizedBox(height: 16),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('最近导出', style: Theme.of(context).textTheme.titleMedium),
                  const SizedBox(height: 8),
                  Text('格式: ${controller.lastExport!.format}'),
                  Text('大小: ${controller.lastExport!.byteSize} bytes'),
                  Text('SHA256: ${controller.lastExport!.sha256}'),
                ],
              ),
            ),
          ),
        ],
      ],
    );
  }
}

class StyleSelector extends StatelessWidget {
  const StyleSelector({super.key, required this.controller});

  final CalligraphyController controller;

  @override
  Widget build(BuildContext context) {
    return SegmentedButton<String>(
      segments: styleOptions.entries
          .map(
            (entry) => ButtonSegment<String>(
              value: entry.key,
              label: Text(entry.value),
            ),
          )
          .toList(),
      selected: {controller.selectedStyle},
      onSelectionChanged: (value) {
        controller.setSelectedStyle(value.single);
      },
    );
  }
}

class PaperSelector extends StatelessWidget {
  const PaperSelector({super.key, required this.controller});

  final CalligraphyController controller;

  @override
  Widget build(BuildContext context) {
    return DropdownButtonFormField<PaperSpec>(
      initialValue: controller.selectedPaper,
      decoration: const InputDecoration(
        labelText: '幅式',
        border: OutlineInputBorder(),
      ),
      items: paperOptions
          .map(
            (paper) => DropdownMenuItem(
              value: paper,
              child: Text(
                '${paper.format} ${paper.widthCm.toStringAsFixed(0)}x${paper.heightCm.toStringAsFixed(0)}cm',
              ),
            ),
          )
          .toList(),
      onChanged: (paper) {
        if (paper == null) {
          return;
        }
        controller.setSelectedPaper(paper);
      },
    );
  }
}

class StatTile extends StatelessWidget {
  const StatTile({
    super.key,
    required this.label,
    required this.value,
    required this.icon,
  });

  final String label;
  final String value;
  final IconData icon;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 150,
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(14),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Icon(icon),
              const SizedBox(height: 8),
              Text(label),
              Text(
                value,
                style: Theme.of(
                  context,
                ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w700),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class GlyphCard extends StatelessWidget {
  const GlyphCard({super.key, required this.glyph, required this.onTap});

  final GlyphSummary glyph;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 150,
      child: Card(
        child: InkWell(
          onTap: onTap,
          child: Padding(
            padding: const EdgeInsets.all(14),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  glyph.character,
                  style: Theme.of(context).textTheme.displaySmall,
                ),
                const SizedBox(height: 8),
                Text(styleOptions[glyph.style] ?? glyph.style),
                Text(glyph.calligrapher),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class GlyphDetailPanel extends StatelessWidget {
  const GlyphDetailPanel({
    super.key,
    required this.detail,
    required this.onPractice,
  });

  final GlyphDetail detail;
  final VoidCallback onPractice;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                ReferenceGlyphCard(glyph: detail.glyph),
                const SizedBox(width: 16),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        '${styleOptions[detail.glyph.style] ?? detail.glyph.style} · ${detail.glyph.calligrapher}',
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                      Text(copybookDisplayName(detail.glyph.copybookId)),
                      const SizedBox(height: 8),
                      const Text('米字格参考'),
                    ],
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            NotesBlock(title: '结构要点', notes: detail.structureNotes),
            NotesBlock(title: '笔法要点', notes: detail.brushworkNotes),
            const SizedBox(height: 12),
            FilledButton.icon(
              onPressed: onPractice,
              icon: const Icon(Icons.draw),
              label: const Text('我已临摹'),
            ),
          ],
        ),
      ),
    );
  }
}

class ReferenceGlyphCard extends StatelessWidget {
  const ReferenceGlyphCard({
    super.key,
    required this.glyph,
    this.size = 132,
    this.fontSize = 76,
  });

  final GlyphSummary glyph;
  final double size;
  final double fontSize;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      label: '临摹参考字 ${glyph.character}',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.center,
        children: [
          Text('临摹参考字', style: Theme.of(context).textTheme.labelLarge),
          const SizedBox(height: 8),
          SizedBox(
            width: size,
            height: size,
            child: CustomPaint(
              painter: ReferenceGlyphPainter(
                character: glyph.character,
                style: Theme.of(context).textTheme.displayLarge,
                fontSize: fontSize,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class ReferenceGlyphPainter extends CustomPainter {
  ReferenceGlyphPainter({
    required this.character,
    required this.style,
    required this.fontSize,
  });

  final String character;
  final TextStyle? style;
  final double fontSize;

  @override
  void paint(Canvas canvas, Size size) {
    final borderPaint = Paint()
      ..color = const Color(0xFF9A6A3A)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1.2;
    final guidePaint = Paint()
      ..color = const Color(0x559A6A3A)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 0.8;

    final rect = Offset.zero & size;
    canvas.drawRect(rect.deflate(0.6), borderPaint);
    canvas.drawLine(
      Offset(size.width / 2, 0),
      Offset(size.width / 2, size.height),
      guidePaint,
    );
    canvas.drawLine(
      Offset(0, size.height / 2),
      Offset(size.width, size.height / 2),
      guidePaint,
    );
    canvas.drawLine(Offset.zero, Offset(size.width, size.height), guidePaint);
    canvas.drawLine(Offset(size.width, 0), Offset(0, size.height), guidePaint);

    final textPainter = TextPainter(
      text: TextSpan(
        text: character,
        style: (style ?? const TextStyle()).copyWith(
          fontSize: fontSize,
          height: 1,
          color: const Color(0xFF1E1B16),
          fontFamily: 'KaiTi',
          fontFamilyFallback: const ['STKaiti', 'serif'],
          fontWeight: FontWeight.w500,
        ),
      ),
      textDirection: TextDirection.ltr,
      textAlign: TextAlign.center,
    )..layout(maxWidth: size.width);
    textPainter.paint(
      canvas,
      Offset(
        (size.width - textPainter.width) / 2,
        (size.height - textPainter.height) / 2,
      ),
    );
  }

  @override
  bool shouldRepaint(covariant ReferenceGlyphPainter oldDelegate) {
    return oldDelegate.character != character ||
        oldDelegate.style != style ||
        oldDelegate.fontSize != fontSize;
  }
}

class NotesBlock extends StatelessWidget {
  const NotesBlock({super.key, required this.title, required this.notes});

  final String title;
  final List<String> notes;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title, style: Theme.of(context).textTheme.titleSmall),
          ...notes.map((note) => Text('· $note')),
        ],
      ),
    );
  }
}

class LayoutPreviewCard extends StatelessWidget {
  const LayoutPreviewCard({super.key, required this.result});

  final LayoutResult result;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              '${result.paper.format} · ${result.columns}列 x ${result.rows}行',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 12),
            AspectRatio(
              aspectRatio: result.paper.widthCm / result.paper.heightCm,
              child: CustomPaint(
                painter: LayoutPreviewPainter(result),
                child: const SizedBox.expand(),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class LayoutPreviewPainter extends CustomPainter {
  LayoutPreviewPainter(this.result);

  final LayoutResult result;

  @override
  void paint(Canvas canvas, Size size) {
    final paperPaint = Paint()..color = const Color(0xfffffbf2);
    final borderPaint = Paint()
      ..color = const Color(0xff8b7a5a)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1;
    canvas.drawRect(Offset.zero & size, paperPaint);
    canvas.drawRect(Offset.zero & size, borderPaint);

    final textPainter = TextPainter(
      textAlign: TextAlign.center,
      textDirection: TextDirection.ltr,
    );
    for (final slot in result.slots) {
      final x = slot.xCm / result.paper.widthCm * size.width;
      final y = slot.yCm / result.paper.heightCm * size.height;
      textPainter.text = TextSpan(
        text: slot.character,
        style: TextStyle(
          color: const Color(0xff1d1b16),
          fontSize: (slot.sizeCm / result.paper.widthCm * size.width).clamp(
            16,
            52,
          ),
          fontWeight: FontWeight.w500,
        ),
      );
      textPainter.layout();
      textPainter.paint(canvas, Offset(x, y));
    }
  }

  @override
  bool shouldRepaint(covariant LayoutPreviewPainter oldDelegate) {
    return oldDelegate.result != result;
  }
}

class EmptyPanel extends StatelessWidget {
  const EmptyPanel({
    super.key,
    required this.icon,
    required this.title,
    required this.message,
  });

  final IconData icon;
  final String title;
  final String message;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          children: [
            Icon(icon, size: 36),
            const SizedBox(height: 8),
            Text(title, style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 4),
            Text(message, textAlign: TextAlign.center),
          ],
        ),
      ),
    );
  }
}
