import 'package:auth_wallet/api_client.dart';
import 'package:auth_wallet/config.dart';
import 'package:auth_wallet/error_alert.dart';
import 'package:auth_wallet/money.dart';
import 'package:auth_wallet/session.dart';
import 'package:auth_wallet/transfer_draft.dart';
import 'package:auth_wallet/transfer_history.dart';
import 'package:auth_wallet/welcome.dart';
import 'package:flutter/material.dart';

void main() {
  final config = ApiConfig.fromEnvironment();
  runApp(
    AuthApp(
      client: ApiClient(baseUrl: config.baseUrl),
      session: Session(),
    ),
  );
}

class AuthApp extends StatefulWidget {
  const AuthApp({super.key, required this.client, required this.session});

  final ApiClient client;
  final Session session;

  @override
  State<AuthApp> createState() => _AuthAppState();
}

enum _Page { login, register, home }

class _AuthAppState extends State<AuthApp> {
  _Page _page = _Page.login;
  String? _notice;

  void _showLogin({String? notice}) {
    setState(() {
      _notice = notice;
      _page = _Page.login;
    });
  }

  void _showHome() {
    setState(() {
      _notice = null;
      _page = _Page.home;
    });
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      home: switch (_page) {
        _Page.register => _RegisterPage(
          client: widget.client,
          onLogin: () => _showLogin(),
          onCreated: () => _showLogin(notice: 'Account created'),
        ),
        _Page.home => _HomePage(
          client: widget.client,
          session: widget.session,
          onLeave: () => _showLogin(),
        ),
        _Page.login => _LoginPage(
          client: widget.client,
          session: widget.session,
          notice: _notice,
          onRegister: () => setState(() => _page = _Page.register),
          onSignedIn: _showHome,
        ),
      },
    );
  }
}

class _LoginPage extends StatefulWidget {
  const _LoginPage({
    required this.client,
    required this.session,
    required this.onRegister,
    required this.onSignedIn,
    this.notice,
  });

  final ApiClient client;
  final Session session;
  final VoidCallback onRegister;
  final VoidCallback onSignedIn;
  final String? notice;

  @override
  State<_LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<_LoginPage> {
  final _email = TextEditingController();
  final _password = TextEditingController();
  String? _error;
  var _busy = false;

  @override
  void dispose() {
    _email.dispose();
    _password.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final token = await widget.client.login(
        email: _email.text,
        password: _password.text,
      );
      if (!mounted) {
        return;
      }
      widget.session.signIn(token: token, email: _email.text);
      widget.onSignedIn();
    } on ApiException catch (err) {
      if (!mounted) {
        return;
      }
      setState(() {
        _busy = false;
        _error = err.message;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Login')),
      body: _FormColumn(
        children: [
          if (widget.notice != null) Text(widget.notice!),
          TextField(
            controller: _email,
            decoration: const InputDecoration(labelText: 'Email'),
            keyboardType: TextInputType.emailAddress,
            autocorrect: false,
          ),
          TextField(
            controller: _password,
            decoration: const InputDecoration(labelText: 'Password'),
            obscureText: true,
          ),
          if (_error != null) ErrorAlert(message: _error!),
          FilledButton(
            onPressed: _busy ? null : _submit,
            child: const Text('Login'),
          ),
          TextButton(
            onPressed: widget.onRegister,
            child: const Text('Create an account'),
          ),
        ],
      ),
    );
  }
}

class _RegisterPage extends StatefulWidget {
  const _RegisterPage({
    required this.client,
    required this.onLogin,
    required this.onCreated,
  });

  final ApiClient client;
  final VoidCallback onLogin;
  final VoidCallback onCreated;

  @override
  State<_RegisterPage> createState() => _RegisterPageState();
}

class _RegisterPageState extends State<_RegisterPage> {
  final _email = TextEditingController();
  final _password = TextEditingController();
  final _confirm = TextEditingController();
  String? _error;
  var _busy = false;

  @override
  void dispose() {
    _email.dispose();
    _password.dispose();
    _confirm.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await widget.client.register(
        email: _email.text,
        password: _password.text,
        confirmPassword: _confirm.text,
      );
      if (!mounted) {
        return;
      }
      widget.onCreated();
    } on ApiException catch (err) {
      if (!mounted) {
        return;
      }
      setState(() {
        _busy = false;
        _error = err.message;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Register')),
      body: _FormColumn(
        children: [
          TextField(
            controller: _email,
            decoration: const InputDecoration(labelText: 'Email'),
            keyboardType: TextInputType.emailAddress,
            autocorrect: false,
          ),
          TextField(
            controller: _password,
            decoration: const InputDecoration(labelText: 'Password'),
            obscureText: true,
          ),
          TextField(
            controller: _confirm,
            decoration: const InputDecoration(labelText: 'Confirm password'),
            obscureText: true,
          ),
          if (_error != null) ErrorAlert(message: _error!),
          FilledButton(
            onPressed: _busy ? null : _submit,
            child: const Text('Register'),
          ),
          TextButton(
            onPressed: widget.onLogin,
            child: const Text('Back to login'),
          ),
        ],
      ),
    );
  }
}

class _HomePage extends StatefulWidget {
  const _HomePage({
    required this.client,
    required this.session,
    required this.onLeave,
  });

  final ApiClient client;
  final Session session;
  final VoidCallback onLeave;

  @override
  State<_HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<_HomePage> with WidgetsBindingObserver {
  final _view = WalletView();
  final _recipient = TextEditingController();
  final _amount = TextEditingController();
  final _notes = TextEditingController();
  late TransferDraft _draft;
  String? _formMessage;
  var _busy = false;
  var _canRetry = false;
  var _resetting = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _draft = TransferDraft();
    _load();
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _recipient.dispose();
    _amount.dispose();
    _notes.dispose();
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed && widget.session.endIfIdle()) {
      widget.onLeave();
    }
  }

  Future<void> _load() async {
    final token = widget.session.token;
    if (token == null) {
      return;
    }
    try {
      final snapshot = await widget.client.wallet(token);
      if (!mounted) {
        return;
      }
      widget.session.completeRequest(succeeded: true);
      setState(() => _view.applyLoad(snapshot));
    } on ApiException {
      if (!mounted) {
        return;
      }
      widget.session.completeRequest(succeeded: false);
      setState(() => _view.applyFailure());
    }
  }

  Future<void> _submit() async {
    final problem = _draft.validate();
    if (problem != null) {
      setState(() => _formMessage = problem);
      return;
    }
    final token = widget.session.token;
    if (token == null) {
      return;
    }
    final draft = _draft;
    setState(() {
      _busy = true;
      _formMessage = null;
    });
    try {
      await widget.client.transfer(
        token: token,
        transferId: draft.transferId,
        recipient: draft.recipient,
        amount: draft.amount,
        notes: draft.notes,
      );
      if (!mounted) {
        return;
      }
      widget.session.completeRequest(succeeded: true);
      setState(_resetDraft);
      await _load();
    } on ApiException catch (err) {
      if (!mounted) {
        return;
      }
      widget.session.completeRequest(succeeded: false);
      setState(() {
        _busy = false;
        _canRetry = true;
        _formMessage = err.message;
      });
    }
  }

  void _resetDraft() {
    _resetting = true;
    _draft = TransferDraft();
    _recipient.clear();
    _amount.clear();
    _notes.clear();
    _resetting = false;
    _busy = false;
    _canRetry = false;
    _formMessage = null;
  }

  void _edit({String? recipient, String? amountText, String? notes}) {
    if (_resetting) {
      return;
    }
    setState(() {
      _draft.edit(recipient: recipient, amountText: amountText, notes: notes);
      _canRetry = false;
    });
  }

  @override
  Widget build(BuildContext context) {
    final email = widget.session.email ?? '';
    final balance = _view.balance;
    return Scaffold(
      appBar: AppBar(
        title: const Text('Wallet'),
        actions: [
          TextButton(
            onPressed: () {
              widget.session.clear();
              widget.onLeave();
            },
            child: const Text('Log out'),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          WelcomeBanner(email: email),
          if (balance != null) ...[
            const SizedBox(height: 8),
            Text(
              formatMinor(balance),
              style: Theme.of(context).textTheme.headlineSmall,
            ),
          ],
          if (_view.error != null) ...[
            const SizedBox(height: 12),
            ErrorAlert(message: _view.error!),
          ],
          if (balance != null) ...[
            const SizedBox(height: 20),
            TransactionHistory(transfers: _view.transfers),
          ],
          const SizedBox(height: 20),
          TextField(
            controller: _recipient,
            decoration: const InputDecoration(labelText: 'Recipient'),
            autocorrect: false,
            onChanged: (value) => _edit(recipient: value),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _amount,
            decoration: const InputDecoration(labelText: 'Amount'),
            keyboardType: const TextInputType.numberWithOptions(decimal: true),
            onChanged: (value) => _edit(amountText: value),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _notes,
            decoration: const InputDecoration(labelText: 'Notes'),
            onChanged: (value) => _edit(notes: value),
          ),
          if (_formMessage != null) ...[
            const SizedBox(height: 12),
            ErrorAlert(message: _formMessage!),
          ],
          const SizedBox(height: 12),
          FilledButton(
            onPressed: _busy ? null : _submit,
            child: const Text('Send'),
          ),
          if (_canRetry)
            TextButton(
              onPressed: _busy ? null : _submit,
              child: const Text('Retry'),
            ),
        ],
      ),
    );
  }
}

class _FormColumn extends StatelessWidget {
  const _FormColumn({required this.children});

  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        for (final child in children) ...[child, const SizedBox(height: 12)],
      ],
    );
  }
}
