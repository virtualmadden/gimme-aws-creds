//! `gac setup`: create an `[sso-session]`, or reuse one that already exists.

use crate::aws_files::{AwsFiles, SsoSessionConfig, managed_block};
use crate::cli::SetupArgs;
use crate::cli::context::Context;
use crate::domain::{Region, SessionName, StartUrl};
use crate::error::{Error, Result};
use crate::ui;

pub async fn run(ctx: &Context, args: SetupArgs) -> Result<()> {
    let files = ctx.files().await?;
    let start_url = args
        .start_url
        .as_deref()
        .map(str::parse::<StartUrl>)
        .transpose()?;
    let region = args
        .region
        .as_deref()
        .map(str::parse::<Region>)
        .transpose()?;

    if let Some(existing) = reusable(ctx, &files, start_url.as_ref(), region.as_ref())? {
        ui::notice(format!(
            "[sso-session {}] already exists (start URL {}, region {}); reusing it.",
            existing.name, existing.start_url, existing.region
        ));
        next_steps(&files, &existing.name, false);
        return Ok(());
    }

    let start_url = match start_url {
        Some(url) => url,
        None => ask(
            ctx,
            "Start URL:",
            None,
            "--start-url",
            "https://my-portal.awsapps.com/start",
        )?,
    };
    let region = match region {
        Some(region) => region,
        None => ask(
            ctx,
            "Identity Center region:",
            None,
            "--region",
            "us-east-1",
        )?,
    };
    let name = if let Some(name) = ctx.session_flag() {
        name.clone()
    } else {
        let suggested = suggest_name(&start_url);
        if ctx.can_prompt() {
            ask(
                ctx,
                "Session name:",
                Some(suggested.as_str()),
                "--session",
                "corp",
            )?
        } else {
            suggested
        }
    };
    if files.has_session(name.as_str()) {
        return Err(Error::config(format!(
            "[sso-session {name}] already exists with different settings"
        ))
        .with_hint("pick another name with --session, or edit the existing section by hand"));
    }

    let text = files.config_text();
    let eol = managed_block::line_ending(text);
    let mut section = String::new();
    for line in [
        format!("[sso-session {name}]"),
        format!("sso_start_url = {start_url}"),
        format!("sso_region = {region}"),
        "sso_registration_scopes = sso:account:access".to_owned(),
    ] {
        section.push_str(&line);
        section.push_str(eol);
    }
    ctx.write_config(text, &managed_block::append(text, &section))
        .await?;
    ui::notice(format!(
        "Added [sso-session {name}] to {}.",
        ctx.paths.config_file().display()
    ));
    next_steps(&files, &name, true);
    Ok(())
}

/// An existing session that matches what was asked for, if any. A named session that exists
/// with different settings is an error: gac never edits a section it did not write.
fn reusable(
    ctx: &Context,
    files: &AwsFiles,
    start_url: Option<&StartUrl>,
    region: Option<&Region>,
) -> Result<Option<SsoSessionConfig>> {
    let matches = |s: &SsoSessionConfig| {
        start_url.is_none_or(|u| *u == s.start_url) && region.is_none_or(|r| *r == s.region)
    };
    if let Some(name) = ctx.session_flag() {
        if !files.has_session(name.as_str()) {
            return Ok(None);
        }
        let existing = files.sso_session(name)?;
        if !matches(&existing) {
            return Err(Error::config(format!(
                "[sso-session {name}] already exists with start URL {} and region {}",
                existing.start_url, existing.region
            ))
            .with_hint("gac does not edit sections it did not write; change it by hand or pick another --session"));
        }
        return Ok(Some(existing));
    }
    let sessions: Vec<SsoSessionConfig> = files
        .session_names()
        .into_iter()
        .filter_map(|name| files.sso_session(&name.parse().ok()?).ok())
        .collect();
    Ok(match (start_url, sessions.as_slice()) {
        (Some(_), _) => sessions.into_iter().find(|s| matches(s)),
        (None, [only]) if region.is_none() => Some(only.clone()),
        _ => None,
    })
}

fn ask<T: std::str::FromStr<Err = Error>>(
    ctx: &Context,
    message: &str,
    default: Option<&str>,
    flag: &str,
    example: &str,
) -> Result<T> {
    if !ctx.can_prompt() {
        return Err(Error::usage(format!(
            "{flag} is required without an interactive terminal"
        ))
        .with_hint(format!("pass {flag} {example}")));
    }
    let answer = ui::prompt::text(message, default, Some(example), |input| {
        input.parse::<T>().map(drop).map_err(|err| err.to_string())
    })?;
    answer.parse()
}

/// The first label of the portal host, e.g. `my-portal` for `https://my-portal.awsapps.com/start`.
fn suggest_name(start_url: &StartUrl) -> SessionName {
    url::Url::parse(start_url.as_str())
        .ok()
        .and_then(|url| {
            url.host_str()
                .map(|h| h.split('.').next().unwrap_or(h).to_owned())
        })
        .and_then(|label| label.parse().ok())
        .unwrap_or_else(|| "default".parse().expect("a valid session name"))
}

fn next_steps(files: &AwsFiles, name: &SessionName, created: bool) {
    let sessions = files.session_names().len() + usize::from(created);
    let flag = if sessions > 1 {
        format!(" --session {name}")
    } else {
        String::new()
    };
    ui::notice(format!("Next: gac login{flag}, then gac sync{flag}"));
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn suggests_the_portal_label() {
        let url: StartUrl = "https://my-portal.awsapps.com/start".parse().unwrap();
        assert_eq!(suggest_name(&url).as_str(), "my-portal");
    }
}
