class Session {
  Session({DateTime Function()? now}) : now = now ?? DateTime.now;

  static const idleLimit = Duration(minutes: 15);

  final DateTime Function() now;
  String? token;
  String? email;
  DateTime? lastSuccess;

  bool get isSignedIn => token != null && email != null;

  void signIn({required String token, required String email, DateTime? at}) {
    this.token = token;
    this.email = email;
    lastSuccess = at ?? now();
  }

  void completeRequest({required bool succeeded}) {
    if (succeeded && isSignedIn) {
      lastSuccess = now();
    }
  }

  bool get isIdle {
    final last = lastSuccess;
    if (last == null) {
      return true;
    }
    return !now().isBefore(last.add(idleLimit));
  }

  bool endIfIdle() {
    if (!isSignedIn || !isIdle) {
      return false;
    }
    clear();
    return true;
  }

  void clear() {
    token = null;
    email = null;
    lastSuccess = null;
  }
}
