import 'package:flutter/material.dart';
import 'app.dart';
import 'config/app_config.dart';

/// Entry point for the Streaks habit tracker.
///
/// CLI arguments (for Linux desktop / testing):
///   --base-url <url>      Mobpilot Identity API base URL
///   --kratos-url <url>    Ory Kratos public URL
///   --hydra-url <url>     Ory Hydra public URL
///   --dev-token <token>   Skip auth, use this bearer token
///   --test-mode           Enable test mode features
///   --client-id <id>      OAuth2 client ID
///   --redirect-uri <uri>  OAuth2 redirect URI
void main(List<String> args) {
  WidgetsFlutterBinding.ensureInitialized();
  final config = AppConfig.fromArgs(args);
  runApp(StreaksApp(config: config));
}
