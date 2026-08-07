import java.util.Base64

plugins {
    id("com.android.application")
    // The Flutter Gradle Plugin must be applied after the Android and Kotlin Gradle plugins.
    id("dev.flutter.flutter-gradle-plugin")
}

val releaseKeystorePath = System.getenv("CALLIGRAPHY_ANDROID_KEYSTORE")
val releaseKeystorePassword = System.getenv("CALLIGRAPHY_ANDROID_KEYSTORE_PASSWORD")
val releaseKeyAlias = System.getenv("CALLIGRAPHY_ANDROID_KEY_ALIAS")
val releaseKeyPassword = System.getenv("CALLIGRAPHY_ANDROID_KEY_PASSWORD")
val releaseSigningConfigured = listOf(
    releaseKeystorePath,
    releaseKeystorePassword,
    releaseKeyAlias,
    releaseKeyPassword,
).all { !it.isNullOrBlank() }
val releaseTaskRequested = gradle.startParameter.taskNames.any {
    it.contains("release", ignoreCase = true)
}
val releaseDartDefines = (project.findProperty("dart-defines") as String?)
    .orEmpty()
    .split(',')
    .filter { it.isNotBlank() }
    .map { Base64.getDecoder().decode(it).toString(Charsets.UTF_8) }
    .filter { it.contains('=') }
    .associate { it.substringBefore('=') to it.substringAfter('=') }

if (releaseTaskRequested && !releaseSigningConfigured) {
    throw GradleException(
        "Release signing requires CALLIGRAPHY_ANDROID_KEYSTORE, " +
            "CALLIGRAPHY_ANDROID_KEYSTORE_PASSWORD, CALLIGRAPHY_ANDROID_KEY_ALIAS, " +
            "and CALLIGRAPHY_ANDROID_KEY_PASSWORD.",
    )
}

if (releaseTaskRequested) {
    val apiBaseUrl = releaseDartDefines["CALLIGRAPHY_API_BASE_URL"].orEmpty()
    if (!apiBaseUrl.startsWith("https://")) {
        throw GradleException(
            "Release builds require an HTTPS CALLIGRAPHY_API_BASE_URL dart define.",
        )
    }
    val oidcClientId = releaseDartDefines["CALLIGRAPHY_OIDC_CLIENT_ID"].orEmpty()
    val oidcRedirectUri = releaseDartDefines["CALLIGRAPHY_OIDC_REDIRECT_URI"].orEmpty()
    if (releaseDartDefines["CALLIGRAPHY_ALLOW_INSECURE_OIDC"] == "true") {
        throw GradleException(
            "Release builds cannot enable CALLIGRAPHY_ALLOW_INSECURE_OIDC.",
        )
    }
    if (!Regex("^[A-Za-z0-9._-]{1,128}$").matches(oidcClientId)) {
        throw GradleException(
            "Release builds require a valid CALLIGRAPHY_OIDC_CLIENT_ID dart define.",
        )
    }
    if (oidcRedirectUri != "com.nebula.calligraphy:/oauthredirect") {
        throw GradleException(
            "Release builds require CALLIGRAPHY_OIDC_REDIRECT_URI=" +
                "com.nebula.calligraphy:/oauthredirect.",
        )
    }
}

android {
    namespace = "com.nebula.calligraphy"
    compileSdk = flutter.compileSdkVersion
    ndkVersion = "28.2.13676358"

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    defaultConfig {
        applicationId = "com.nebula.calligraphy"
        minSdk = 24
        targetSdk = flutter.targetSdkVersion
        versionCode = flutter.versionCode
        versionName = flutter.versionName
        manifestPlaceholders.putAll(
            mapOf("appAuthRedirectScheme" to "com.nebula.calligraphy"),
        )
    }

    if (releaseSigningConfigured) {
        signingConfigs {
            create("release") {
                storeFile = file(releaseKeystorePath!!)
                storePassword = releaseKeystorePassword
                keyAlias = releaseKeyAlias
                keyPassword = releaseKeyPassword
                enableV1Signing = true
                enableV2Signing = true
                enableV3Signing = true
                enableV4Signing = true
            }
        }
    }

    buildTypes {
        getByName("release") {
            if (releaseSigningConfigured) {
                signingConfig = signingConfigs.getByName("release")
            }
            isMinifyEnabled = true
            isShrinkResources = true
        }
    }
}

kotlin {
    compilerOptions {
        jvmTarget = org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17
    }
}

flutter {
    source = "../.."
}
