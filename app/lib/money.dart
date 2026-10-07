class MoneyParseException implements Exception {
  MoneyParseException(this.message);

  final String message;

  @override
  String toString() => message;
}

String formatMinor(int minor) {
  final negative = minor < 0;
  final abs = minor.abs();
  final whole = abs ~/ 100;
  final frac = (abs % 100).toString().padLeft(2, '0');
  final text = '$whole.$frac';
  return negative ? '-$text' : text;
}

int parseAmount(String text) {
  final match = RegExp(r'^(\d+)(?:\.(\d{1,2}))?$').firstMatch(text.trim());
  if (match == null) {
    throw MoneyParseException(
      'amount must be a positive number with at most two decimal places',
    );
  }
  final whole = int.parse(match.group(1)!);
  final fracRaw = match.group(2) ?? '';
  final frac = int.parse(fracRaw.padRight(2, '0'));
  final minor = whole * 100 + frac;
  if (minor <= 0) {
    throw MoneyParseException('amount must be positive');
  }
  return minor;
}
