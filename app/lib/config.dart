class ApiConfig {
  const ApiConfig({required this.baseUrl});

  factory ApiConfig.fromEnvironment() {
    const defined = String.fromEnvironment('API_BASE_URL');
    final trimmed = defined.trim();
    final base = trimmed.isEmpty ? 'http://10.0.2.2:8080' : trimmed;
    final slash = base.endsWith('/') ? base.substring(0, base.length - 1) : base;
    return ApiConfig(baseUrl: slash);
  }

  final String baseUrl;
}
