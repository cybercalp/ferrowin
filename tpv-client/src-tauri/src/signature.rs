use sha2::{Sha256, Digest};

pub trait Firmador {
    fn firmar_registro(
        &self,
        prefijo: &str,
        secuencia: i64,
        total: f64,
        creado_en: &str,
        hash_anterior: Option<&str>,
    ) -> Result<String, String>;
}

pub struct FirmaSimulada;

impl Firmador for FirmaSimulada {
    fn firmar_registro(
        &self,
        prefijo: &str,
        secuencia: i64,
        total: f64,
        creado_en: &str,
        hash_anterior: Option<&str>,
    ) -> Result<String, String> {
        let input = format!(
            "{}:{}:{}:{}:{}",
            prefijo,
            secuencia,
            total,
            creado_en,
            hash_anterior.unwrap_or("")
        );

        let mut hasher = Sha256::new();
        hasher.update(input.as_bytes());
        let result = hasher.finalize();

        let hash_hex = result
            .iter()
            .map(|b| format!("{:02x}", b))
            .collect::<String>();

        Ok(hash_hex)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_firma_simulada_varying_inputs() {
        let firmador = FirmaSimulada;
        
        let base_prefijo = "TPV";
        let base_secuencia = 1;
        let base_total = 10.50;
        let base_creado = "2026-06-05T10:00:00Z";
        let base_hash_anterior = Some("hash_previo");

        let base_hash = firmador.firmar_registro(base_prefijo, base_secuencia, base_total, base_creado, base_hash_anterior).unwrap();

        // 1. Varying prefijo
        let hash_diff_prefijo = firmador.firmar_registro("TPV2", base_secuencia, base_total, base_creado, base_hash_anterior).unwrap();
        assert_ne!(base_hash, hash_diff_prefijo);

        // 2. Varying secuencia
        let hash_diff_secuencia = firmador.firmar_registro(base_prefijo, 2, base_total, base_creado, base_hash_anterior).unwrap();
        assert_ne!(base_hash, hash_diff_secuencia);

        // 3. Varying total
        let hash_diff_total = firmador.firmar_registro(base_prefijo, base_secuencia, 11.50, base_creado, base_hash_anterior).unwrap();
        assert_ne!(base_hash, hash_diff_total);

        // 4. Varying creado_en
        let hash_diff_creado = firmador.firmar_registro(base_prefijo, base_secuencia, base_total, "2026-06-05T10:00:01Z", base_hash_anterior).unwrap();
        assert_ne!(base_hash, hash_diff_creado);

        // 5. Varying hash_anterior
        let hash_diff_anterior = firmador.firmar_registro(base_prefijo, base_secuencia, base_total, base_creado, Some("hash_previo_different")).unwrap();
        assert_ne!(base_hash, hash_diff_anterior);
    }

    #[test]
    fn test_firma_simulada_empty_anterior() {
        let firmador = FirmaSimulada;
        let hash_none = firmador.firmar_registro("TPV", 1, 10.50, "2026-06-05T10:00:00Z", None).unwrap();
        let hash_empty = firmador.firmar_registro("TPV", 1, 10.50, "2026-06-05T10:00:00Z", Some("")).unwrap();
        
        // They should be identical since unwrap_or("") is used
        assert_eq!(hash_none, hash_empty);
    }
}

