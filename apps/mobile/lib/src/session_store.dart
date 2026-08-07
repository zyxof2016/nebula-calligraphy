import 'dart:convert';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import 'models.dart';

class StoredSession {
  const StoredSession({
    required this.apiBaseUrl,
    required this.token,
    required this.expiresAt,
    required this.user,
  });

  factory StoredSession.fromJson(Map<String, dynamic> json) {
    return StoredSession(
      apiBaseUrl: json['api_base_url']?.toString() ?? '',
      token: json['token']?.toString() ?? '',
      expiresAt: json['expires_at']?.toString() ?? '',
      user: User.fromJson(json['user'] as Map<String, dynamic>),
    );
  }

  Map<String, dynamic> toJson() => {
    'api_base_url': apiBaseUrl,
    'token': token,
    'expires_at': expiresAt,
    'user': user.toJson(),
  };

  final String apiBaseUrl;
  final String token;
  final String expiresAt;
  final User user;
}

abstract class SessionStore {
  Future<StoredSession?> load();

  Future<void> save(StoredSession session);

  Future<void> clear();
}

class MemorySessionStore implements SessionStore {
  StoredSession? _session;

  @override
  Future<StoredSession?> load() async => _session;

  @override
  Future<void> save(StoredSession session) async {
    _session = session;
  }

  @override
  Future<void> clear() async {
    _session = null;
  }
}

class SecureSessionStore implements SessionStore {
  static const _key = 'nebula_calligraphy_session';
  static const _storage = FlutterSecureStorage(
    aOptions: AndroidOptions(storageNamespace: 'nebula_calligraphy'),
    iOptions: IOSOptions(
      accessibility: KeychainAccessibility.first_unlock_this_device,
    ),
  );

  @override
  Future<StoredSession?> load() async {
    final raw = await _storage.read(key: _key);
    if (raw == null || raw.isEmpty) {
      return null;
    }
    try {
      return StoredSession.fromJson(jsonDecode(raw) as Map<String, dynamic>);
    } catch (_) {
      await _storage.delete(key: _key);
      return null;
    }
  }

  @override
  Future<void> save(StoredSession session) async {
    await _storage.write(key: _key, value: jsonEncode(session.toJson()));
  }

  @override
  Future<void> clear() async {
    await _storage.delete(key: _key);
  }
}
