import 'package:auth_wallet/api_client.dart';
import 'package:auth_wallet/money.dart';
import 'package:flutter/material.dart';

/// The signed-in list of transfers this account sent or received.
class TransactionHistory extends StatelessWidget {
  const TransactionHistory({super.key, required this.transfers});

  final List<TransferEntry> transfers;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text('History', style: Theme.of(context).textTheme.titleMedium),
        const SizedBox(height: 8),
        if (transfers.isEmpty)
          const Text('No transfers yet')
        else
          for (var i = 0; i < transfers.length; i++) ...[
            if (i > 0) const SizedBox(height: 8),
            TransactionTile(entry: transfers[i]),
          ],
      ],
    );
  }
}

/// One sent or received transfer, including its notes.
class TransactionTile extends StatelessWidget {
  const TransactionTile({super.key, required this.entry});

  final TransferEntry entry;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final received = entry.direction == 'received';
    final label = switch (entry.direction) {
      'received' => 'Received',
      'sent' => 'Sent',
      _ => entry.direction,
    };
    final foreground = received ? scheme.onSecondaryContainer : scheme.error;
    final background = received
        ? scheme.secondaryContainer
        : scheme.errorContainer;
    final details = [
      '$label ${formatMinor(entry.amount)}',
      entry.counterparty,
      if (entry.notes.isNotEmpty) entry.notes,
    ].join(', ');

    return Semantics(
      container: true,
      label: details,
      child: Card(
        margin: EdgeInsets.zero,
        color: background,
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              ExcludeSemantics(
                child: Icon(
                  received ? Icons.call_received : Icons.call_made,
                  color: foreground,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      '$label ${formatMinor(entry.amount)}',
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        color: foreground,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      entry.counterparty,
                      style: Theme.of(context).textTheme.bodyMedium
                          ?.copyWith(color: scheme.onSurface),
                    ),
                    if (entry.notes.isNotEmpty) ...[
                      const SizedBox(height: 4),
                      Text(
                        entry.notes,
                        style: Theme.of(context).textTheme.bodyMedium
                            ?.copyWith(color: scheme.onSurfaceVariant),
                      ),
                    ],
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
