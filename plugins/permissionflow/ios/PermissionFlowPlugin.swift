import Foundation
import ObjectiveC
import UIKit
import UserNotifications

/// The system's permissions, by name: a binary that names the frameworks' classes
/// has to declare why it would use them, which an app that asks for one of them
/// should not have to for all.
final class PermissionFlowPlugin: MyGoPlugin {
    private func load(_ framework: String) {
        _ = Bundle(path: "/System/Library/Frameworks/\(framework).framework")?.load()
    }

    private typealias StatusFn = @convention(c) (AnyClass, Selector, AnyObject) -> Int
    private typealias IntStatusFn = @convention(c) (AnyClass, Selector, Int) -> Int
    private typealias AskFn = @convention(c) (AnyClass, Selector, AnyObject, @convention(block) (Bool) -> Void) -> Void

    private func avStatus(_ media: String) -> String {
        load("AVFoundation")
        guard let c = NSClassFromString("AVCaptureDevice") else { return "unknown" }
        let sel = NSSelectorFromString("authorizationStatusForMediaType:")
        guard let m = class_getClassMethod(c, sel) else { return "unknown" }
        let fn = unsafeBitCast(method_getImplementation(m), to: StatusFn.self)
        switch fn(c, sel, media as NSString) {
        case 0: return "notDetermined"
        case 3: return "granted"
        default: return "denied"
        }
    }

    private func avRequest(_ media: String, _ done: @escaping (Bool) -> Void) {
        load("AVFoundation")
        let sel = NSSelectorFromString("requestAccessForMediaType:completionHandler:")
        guard let c = NSClassFromString("AVCaptureDevice"), let m = class_getClassMethod(c, sel) else { return done(false) }
        let fn = unsafeBitCast(method_getImplementation(m), to: AskFn.self)
        fn(c, sel, media as NSString, { done($0) })
    }

    private func photosStatus() -> String {
        load("Photos")
        let sel = NSSelectorFromString("authorizationStatusForAccessLevel:")
        guard let c = NSClassFromString("PHPhotoLibrary"), let m = class_getClassMethod(c, sel) else { return "unknown" }
        let fn = unsafeBitCast(method_getImplementation(m), to: IntStatusFn.self)
        switch fn(c, sel, 2) { // read and write
        case 0: return "notDetermined"
        case 3, 4: return "granted" // authorized, or limited
        default: return "denied"
        }
    }

    private func photosRequest(_ done: @escaping (Bool) -> Void) {
        load("Photos")
        typealias Fn = @convention(c) (AnyClass, Selector, Int, @convention(block) (Int) -> Void) -> Void
        let sel = NSSelectorFromString("requestAuthorizationForAccessLevel:handler:")
        guard let c = NSClassFromString("PHPhotoLibrary"), let m = class_getClassMethod(c, sel) else { return done(false) }
        let fn = unsafeBitCast(method_getImplementation(m), to: Fn.self)
        fn(c, sel, 2, { done($0 == 3 || $0 == 4) })
    }

    private func contactsStatus() -> String {
        load("Contacts")
        let sel = NSSelectorFromString("authorizationStatusForEntityType:")
        guard let c = NSClassFromString("CNContactStore"), let m = class_getClassMethod(c, sel) else { return "unknown" }
        let fn = unsafeBitCast(method_getImplementation(m), to: IntStatusFn.self)
        switch fn(c, sel, 0) {
        case 0: return "notDetermined"
        case 3, 4: return "granted"
        default: return "denied"
        }
    }

    private func contactsRequest(_ done: @escaping (Bool) -> Void) {
        load("Contacts")
        typealias Fn = @convention(c) (AnyObject, Selector, Int, @convention(block) (Bool, NSError?) -> Void) -> Void
        let sel = NSSelectorFromString("requestAccessForEntityType:completionHandler:")
        guard let c = NSClassFromString("CNContactStore") as? NSObject.Type, let m = class_getInstanceMethod(c, sel) else { return done(false) }
        let store = c.init()
        let fn = unsafeBitCast(method_getImplementation(m), to: Fn.self)
        fn(store, sel, 0, { ok, _ in _ = store; done(ok) })
    }

    private func usage(_ key: String) -> Bool { Bundle.main.object(forInfoDictionaryKey: key) != nil }

    private func status(_ p: String, _ done: @escaping (String?) -> Void) {
        switch p {
        case "camera": done(avStatus("vide"))
        case "microphone": done(avStatus("soun"))
        case "photos": done(photosStatus())
        case "contacts": done(contactsStatus())
        case "notifications":
            UNUserNotificationCenter.current().getNotificationSettings { s in
                DispatchQueue.main.async {
                    switch s.authorizationStatus {
                    case .notDetermined: done("notDetermined")
                    case .denied: done("denied")
                    default: done("granted")
                    }
                }
            }
        default: done(nil)
        }
    }

    func call(_ method: String, _ args: [String: Any], _ reply: @escaping PluginReply) {
        let p = args["permission"] as? String ?? ""
        let answer: (String?) -> Void = { s in
            if let s {
                reply(.success(["status": s]))
            } else {
                reply(.failure(PluginFailure("Unsupported")))
            }
        }
        switch method {
        case "check":
            status(p, answer)
        case "request":
            let after: (Bool) -> Void = { _ in DispatchQueue.main.async { self.status(p, answer) } }
            let need = ["camera": "NSCameraUsageDescription", "microphone": "NSMicrophoneUsageDescription",
                        "photos": "NSPhotoLibraryUsageDescription", "contacts": "NSContactsUsageDescription"][p]
            if let need, !usage(need) {
                reply(.failure(PluginFailure("NoUsageDescription", need)))
                return
            }
            switch p {
            case "camera": avRequest("vide", after)
            case "microphone": avRequest("soun", after)
            case "photos": photosRequest(after)
            case "contacts": contactsRequest(after)
            case "notifications":
                UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .sound, .badge]) { ok, _ in after(ok) }
            default: answer(nil)
            }
        case "openSettings":
            if let u = URL(string: UIApplication.openSettingsURLString) { UIApplication.shared.open(u) }
            reply(.success(nil))
        default:
            reply(.failure(PluginFailure("NoMethod", method)))
        }
    }
}
