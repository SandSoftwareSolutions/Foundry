//! {{PROJECT_NAME}}: a small command line program.

use std::env;

/// Builds the greeting. Kept separate from `main` so it is easy to test.
fn greeting(name: &str) -> String {
    format!("Hello, {}! This is {{PROJECT_NAME}}.", name)
}

fn main() {
    let name = env::args().nth(1).unwrap_or_else(|| "world".to_string());
    println!("{}", greeting(&name));
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn greets_a_name() {
        assert_eq!(greeting("Ada"), "Hello, Ada! This is {{PROJECT_NAME}}.");
    }
}
