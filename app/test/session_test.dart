import 'package:auth_wallet/config.dart';
import 'package:auth_wallet/session.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final start = DateTime.utc(2026, 10, 7, 12);

  Session sessionAt(DateTime time) {
    return Session(now: () => time);
  }

  test('default API base URL is the Android emulator host', () {
    expect(ApiConfig.fromEnvironment().baseUrl, 'http://10.0.2.2:8080');
  });

  test('resume at 14 minutes keeps the session', () {
    final session = sessionAt(start);
    session.signIn(token: 'token', email: 'ada@example.com', at: start);
    final later = Session(now: () => start.add(const Duration(minutes: 14)));
    later.token = session.token;
    later.email = session.email;
    later.lastSuccess = session.lastSuccess;

    expect(later.endIfIdle(), isFalse);
    expect(later.isSignedIn, isTrue);
    expect(later.email, 'ada@example.com');
  });

  test('resume at 15 minutes clears the session without a network call', () {
    final session = Session(now: () => start.add(const Duration(minutes: 15)));
    session.signIn(token: 'token', email: 'ada@example.com', at: start);

    expect(session.endIfIdle(), isTrue);
    expect(session.isSignedIn, isFalse);
    expect(session.token, isNull);
  });

  test('a failed call leaves the stored time unchanged', () {
    var current = start;
    final session = Session(now: () => current);
    session.signIn(token: 'token', email: 'ada@example.com', at: start);
    final stored = session.lastSuccess;

    current = start.add(const Duration(minutes: 10));
    session.completeRequest(succeeded: false);

    expect(session.lastSuccess, stored);
    current = start.add(const Duration(minutes: 15));
    expect(session.endIfIdle(), isTrue);
  });

  test('a successful call extends the window', () {
    var current = start;
    final session = Session(now: () => current);
    session.signIn(token: 'token', email: 'ada@example.com', at: start);

    current = start.add(const Duration(minutes: 14));
    session.completeRequest(succeeded: true);
    expect(session.lastSuccess, current);

    current = current.add(const Duration(minutes: 14));
    expect(session.endIfIdle(), isFalse);
    expect(session.isSignedIn, isTrue);
  });
}
